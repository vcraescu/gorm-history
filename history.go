package history

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"time"
)

const (
	ActionCreate    Action             = "create"
	ActionUpdate    Action             = "update"
	userOptionKey   userOptionCtxKey   = pluginName + ":user"
	sourceOptionKey sourceOptionCtxKey = pluginName + ":source"
)

var (
	_ History              = (*Entry)(nil)
	_ TimestampableHistory = (*Entry)(nil)
	_ BlameableHistory     = (*Entry)(nil)
	_ SourceableHistory    = (*Entry)(nil)
)

type (
	Version string

	Action string

	userOptionCtxKey string

	sourceOptionCtxKey string

	Recordable interface {
		CreateHistory() History
	}

	TimestampableHistory interface {
		SetHistoryCreatedAt(createdAt time.Time)
	}

	BlameableHistory interface {
		SetHistoryUserID(id string)
		SetHistoryUserEmail(email string)
	}

	SourceableHistory interface {
		SetHistorySourceID(ID string)
		SetHistorySourceType(typ string)
	}

	History interface {
		SetHistoryVersion(version Version)
		SetHistoryObjectID(id interface{})
		SetHistoryAction(action Action)
	}

	Entry struct {
		Version    Version   `gorm:"type:char(26)"`
		ObjectID   string    `gorm:"index"`
		Action     Action    `gorm:"type:varchar(24)"`
		UserID     string    `gorm:"type:varchar(255)"`
		UserEmail  string    `gorm:"type:varchar(255)"`
		SourceID   string    `gorm:"type:varchar(255)"`
		SourceType string    `gorm:"type:varchar(255)"`
		CreatedAt  time.Time `gorm:"type:datetime"`
	}

	User struct {
		ID    string
		Email string
	}

	Source struct {
		ID   string
		Type string
	}
)

func SetUser(db *gorm.DB, user User) *gorm.DB {
	ctx := context.WithValue(db.Statement.Context, userOptionKey, user)

	return db.WithContext(ctx).Set(string(userOptionKey), user)
}

func GetUser(db *gorm.DB) (User, bool) {
	value, ok := db.Get(string(userOptionKey))
	if !ok {
		value := db.Statement.Context.Value(userOptionKey)
		user, ok := value.(User)

		return user, ok
	}

	user, ok := value.(User)

	return user, ok
}

func SetSource(db *gorm.DB, source Source) *gorm.DB {
	ctx := context.WithValue(db.Statement.Context, sourceOptionKey, source)

	return db.WithContext(ctx).Set(string(sourceOptionKey), source)
}

func GetSource(db *gorm.DB) (Source, bool) {
	value, ok := db.Get(string(sourceOptionKey))
	if !ok {
		value := db.Statement.Context.Value(sourceOptionKey)
		source, ok := value.(Source)

		return source, ok
	}

	source, ok := value.(Source)

	return source, ok
}

func (e *Entry) SetHistoryVersion(version Version) {
	e.Version = version
}

func (e *Entry) SetHistoryObjectID(id interface{}) {
	e.ObjectID = fmt.Sprintf("%v", id)
}

func (e *Entry) SetHistoryAction(action Action) {
	e.Action = action
}

func (e *Entry) SetHistoryUserID(id string) {
	e.UserID = id
}

func (e *Entry) SetHistoryUserEmail(email string) {
	e.UserEmail = email
}

func (e *Entry) SetHistoryCreatedAt(createdAt time.Time) {
	e.CreatedAt = createdAt
}

func (e *Entry) SetHistorySourceID(id string) {
	e.SourceID = id
}

func (e *Entry) SetHistorySourceType(typ string) {
	e.SourceType = typ
}
