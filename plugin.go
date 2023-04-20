package history

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/jinzhu/copier"
)

const (
	pluginName                             = "gorm-history"
	createCbName                           = pluginName + ":after_create"
	updateCbName                           = pluginName + ":after_update"
	disabledOptionKey disabledOptionCtxKey = pluginName + ":disabled"
)

var (
	_ gorm.Plugin = (*Plugin)(nil)

	ErrUnsupportedOperation = errors.New("history is not supported for this operation")
)

type (
	disabledOptionCtxKey string

	VersionFunc func(ctx *Context) (Version, error)

	CopyFunc func(r Recordable, h interface{}) error

	callback func(db *gorm.DB)

	Context struct {
		object   Recordable
		history  History
		objectID interface{}
		action   Action
		db       *gorm.DB
	}

	Option struct{}

	Config struct {
		VersionFunc VersionFunc
		CopyFunc    CopyFunc
	}

	ConfigFunc func(c *Config)

	ULIDVersion struct {
		entropy io.Reader
		mu      sync.Mutex
	}

	IsZeroer interface {
		IsZero() bool
	}

	primaryKeyField struct {
		name   string
		value  interface{}
		isZero bool
	}

	Plugin struct {
		versionFunc VersionFunc
		copyFunc    CopyFunc
		createCb    callback
		updateCb    callback
	}
)

func New(configFuncs ...ConfigFunc) *Plugin {
	version := NewULIDVersion()
	cfg := &Config{
		VersionFunc: version.Version,
		CopyFunc:    DefaultCopyFunc,
	}

	for _, f := range configFuncs {
		f(cfg)
	}

	p := Plugin{
		versionFunc: cfg.VersionFunc,
		copyFunc:    cfg.CopyFunc,
	}

	return &p
}

func WithVersionFunc(fn VersionFunc) ConfigFunc {
	return func(c *Config) {
		c.VersionFunc = fn
	}
}

func WithCopyFunc(fn CopyFunc) ConfigFunc {
	return func(c *Config) {
		c.CopyFunc = fn
	}
}

func NewULIDVersion() *ULIDVersion {
	entropy := ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

	return &ULIDVersion{entropy: entropy}
}

func Disable(db *gorm.DB) *gorm.DB {
	ctx := context.WithValue(db.Statement.Context, disabledOptionKey, true)

	return db.WithContext(ctx).Set(string(disabledOptionKey), true)
}

func IsDisabled(db *gorm.DB) bool {
	_, ok := db.Get(string(disabledOptionKey))
	if ok {
		return true
	}

	return db.Statement.Context.Value(disabledOptionKey) != nil
}

func (p *Plugin) Name() string {
	return pluginName
}

func (p *Plugin) Initialize(db *gorm.DB) error {
	p.createCb = p.callback(ActionCreate)
	p.updateCb = p.callback(ActionUpdate)

	err := db.
		Callback().
		Create().
		After("gorm:create").
		Register(createCbName, p.createCb)
	if err != nil {
		return err
	}

	return db.
		Callback().
		Update().
		After("gorm:update").
		Register(updateCbName, p.updateCb)
}

func (p Plugin) callback(action Action) func(db *gorm.DB) {
	return func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}

		if IsDisabled(db) {
			return
		}

		v := db.Statement.ReflectValue

		switch v.Kind() {
		case reflect.Struct:
			h, isRecordable, err := p.processStruct(v, action, db)
			if err != nil {
				db.AddError(err)
				return
			}

			if !isRecordable {
				return
			}

			if err := p.saveHistory(db, h); err != nil {
				db.AddError(err)
				return
			}
		case reflect.Slice:
			hs, err := p.processSlice(v, action, db)
			if err != nil {
				db.AddError(err)
				return
			}

			if len(hs) == 0 {
				return
			}

			if err := p.saveHistory(db, hs...); err != nil {
				db.AddError(err)
				return
			}
		}
	}
}

func (p *Plugin) saveHistory(db *gorm.DB, hs ...History) error {
	if len(hs) == 0 {
		return nil
	}

	db = db.Session(&gorm.Session{
		NewDB: true,
	})
	for _, h := range hs {
		if err := db.Omit(clause.Associations).Create(h).Error; err != nil {
			return err
		}
	}

	return nil
}

func (p *Plugin) processStruct(v reflect.Value, action Action, db *gorm.DB) (History, bool, error) {
	vi := v.Interface()
	r, ok := vi.(Recordable)
	if !ok {
		return nil, false, nil
	}

	pk, err := getPrimaryKeyValue(db, v)
	if err != nil {
		return nil, false, err
	}

	if pk.isZero {
		return nil, false, fmt.Errorf("not able to determine record primary key value: %w", ErrUnsupportedOperation)
	}

	h, err := p.newHistory(r, action, db, pk)
	if err != nil {
		return nil, true, err
	}

	return h, true, nil
}

func (p *Plugin) processSlice(v reflect.Value, action Action, db *gorm.DB) ([]History, error) {
	var hs []History
	for i := 0; i < v.Len(); i++ {
		el := v.Index(i)

		h, isRecordable, err := p.processStruct(el, action, db)
		if err != nil {
			return nil, err
		}

		if !isRecordable {
			continue
		}

		hs = append(hs, h)
	}

	return hs, nil
}

func (p *Plugin) newHistory(r Recordable, action Action, db *gorm.DB, pk *primaryKeyField) (History, error) {
	hist := r.CreateHistory()
	ihist := makePtr(hist)
	if err := p.copyFunc(r, ihist); err != nil {
		return nil, err
	}

	if err := unsetStructField(hist, pk.name); err != nil {
		return nil, err
	}

	if err := db.Statement.Parse(hist); err != nil {
		return nil, err
	}

	ctx := &Context{
		object:   r,
		objectID: pk.value,
		history:  hist.(History),
		action:   action,
		db:       db,
	}
	version, err := p.versionFunc(ctx)
	if err != nil {
		return nil, fmt.Errorf("error generating history version: %w", err)
	}

	hist.SetHistoryAction(action)
	hist.SetHistoryVersion(version)
	hist.SetHistoryObjectID(pk.value)

	if th, ok := hist.(TimestampableHistory); ok {
		th.SetHistoryCreatedAt(db.NowFunc())
	}

	if bh, ok := hist.(BlameableHistory); ok {
		if user, ok := GetUser(db); ok {
			bh.SetHistoryUserID(user.ID)
			bh.SetHistoryUserEmail(user.Email)
		}
	}

	if bh, ok := hist.(SourceableHistory); ok {
		if source, ok := GetSource(db); ok {
			bh.SetHistorySourceID(source.ID)
			bh.SetHistorySourceType(source.Type)
		}
	}

	return hist.(History), nil
}

func (c *Context) Object() Recordable {
	return c.object
}

func (c *Context) History() History {
	return c.history
}

func (c *Context) ObjectID() interface{} {
	return c.objectID
}

func (c *Context) Action() Action {
	return c.action
}

func (c *Context) DB() *gorm.DB {
	return c.db
}

func (v *ULIDVersion) Version(ctx *Context) (Version, error) {
	if ctx.Action() == ActionCreate {
		return "", nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	uid, err := ulid.New(ulid.Timestamp(time.Now()), v.entropy)
	if err != nil {
		return "", err
	}

	return Version(uid.String()), nil
}

func DefaultCopyFunc(r Recordable, h interface{}) error {
	if reflect.ValueOf(h).Kind() != reflect.Ptr {
		return fmt.Errorf("pointer expected but got %T", h)
	}

	return copier.Copy(h, r)
}
