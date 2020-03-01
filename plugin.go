package history

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"time"

	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
)

const (
	Tag              = "gorm-history"
	FieldTagObjectID = "objectID"
	FieldTagVersion  = "version"
	FieldTagAction   = "action"
	createCbName     = Tag + ":after_create"
	updateCbName     = Tag + ":after_update"
)

type VersionFunc func(context Context) (Version, error)

type CopyFunc func(r Recordable, h interface{}) error

type callback func(scope *gorm.Scope)

type Context struct {
	Object  Recordable
	History interface{}
	Action  Action
	Scope   *gorm.Scope
}

type Option struct {
}

type Config struct {
	VersionFunc VersionFunc
	CopyFunc    CopyFunc
}

type ConfigFunc func(c *Config)

type Plugin struct {
	db          *gorm.DB
	VersionFunc VersionFunc
	CopyFunc    CopyFunc
	createCb    callback
	updateCb    callback
}

func Register(db *gorm.DB, configFuncs ...ConfigFunc) Plugin {
	cfg := &Config{
		VersionFunc: TimedVersionFunc,
		CopyFunc:    DefaultCopyFunc,
	}

	for _, f := range configFuncs {
		f(cfg)
	}

	p := Plugin{
		db:          db,
		VersionFunc: cfg.VersionFunc,
		CopyFunc:    cfg.CopyFunc,
	}

	p.Register()

	return p
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

func (p Plugin) Register() {
	p.createCb = p.callback(ActionCreate)
	p.updateCb = p.callback(ActionUpdate)

	p.db.
		Callback().
		Create().
		After("gorm:create").
		Replace(createCbName, p.createCb)

	p.db.
		Callback().
		Update().
		After("gorm:update").
		Replace(updateCbName, p.updateCb)
}

func (p Plugin) callback(action Action) func(scope *gorm.Scope) {
	return func(scope *gorm.Scope) {
		v := scope.IndirectValue().Addr().Interface()
		if isHistory(v) {
			return
		}

		r, ok := v.(Recordable)
		if !ok {
			return
		}

		hi := r.CreateHistory()
		if reflect.ValueOf(hi).Kind() != reflect.Ptr {
			ptrHi := reflect.New(reflect.TypeOf(hi))
			ptrHi.Elem().Set(reflect.ValueOf(hi))
			hi = ptrHi.Interface()
		}

		if err := p.CopyFunc(r, hi); err != nil {
			panic(err)
		}

		ns := scope.New(hi)
		pk := ns.PrimaryField().Field
		pk.Set(reflect.Zero(pk.Type()))

		id := scope.PrimaryKeyValue()
		ctx := Context{
			Object:  r,
			History: hi,
			Action:  action,
			Scope:   scope,
		}
		version, err := p.VersionFunc(ctx)
		if err != nil {
			panic(err)
		}

		if err := setHistoryFields(hi, id, version, action); err != nil {
			panic(err)
		}

		err = scope.NewDB().Save(hi).Error
		if err != nil {
			panic(err)
		}
	}
}

func TimedVersionFunc(ctx Context) (Version, error) {
	if ctx.Action == ActionCreate {
		return 0, nil
	}

	r, _ := rand.Int(rand.Reader, big.NewInt(5))
	time.Sleep(time.Nanosecond * time.Duration(r.Int64()))

	version := Version(time.Now().UnixNano())

	return version, nil
}

func IncrementedVersionFunc(ctx Context) (Version, error) {
	if ctx.Action == ActionCreate {
		return 0, nil
	}

	db := ctx.Scope.DB()
	scope := ctx.Scope.New(ctx.History)
	tn := scope.TableName()
	fields := GetHistoryFields(ctx.History)
	var versionDBName, objectIDDBName string
	for _, sf := range fields {
		field, ok := scope.FieldByName(sf.Name)
		if !ok {
			continue
		}

		tagValue, ok := sf.Tag.Lookup(Tag)
		if !ok {
			continue
		}

		switch tagValue {
		case FieldTagVersion:
			versionDBName = field.DBName
			break
		case FieldTagObjectID:
			objectIDDBName = field.DBName
		}
	}

	if versionDBName == "" {
		return 0, fmt.Errorf(`field "%s" not defined`, FieldTagVersion)
	}

	if objectIDDBName == "" {
		return 0, fmt.Errorf(`field "%s" not defined`, objectIDDBName)
	}

	var max int
	err := db.
		Table(tn).
		Select(fmt.Sprintf("max(%s)", versionDBName)).
		Where(fmt.Sprintf("%s = ?", objectIDDBName), ctx.Scope.PrimaryKeyValue()).
		Row().
		Scan(&max)
	if err != nil {
		return 0, err
	}

	return Version(max + 1), err
}

func DefaultCopyFunc(r Recordable, h interface{}) error {
	if reflect.ValueOf(h).Kind() != reflect.Ptr {
		return fmt.Errorf("pointer expected but got %T", h)
	}

	return copier.Copy(h, r)
}
