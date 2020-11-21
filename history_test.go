package history_test

import (
	"context"
	"github.com/stretchr/testify/require"
	history "github.com/vcraescu/gorm-history/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestSetUser(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	db = history.SetUser(db, history.User{ID: "foobar"})
	user, ok := history.GetUser(db)
	require.True(t, ok)
	require.Equal(t, "foobar", user.ID)

	db = db.Session(&gorm.Session{})
	user, ok = history.GetUser(db)
	require.True(t, ok)
	require.Equal(t, "foobar", user.ID)

	db = db.WithContext(context.Background())
	user, ok = history.GetUser(db)
	require.True(t, ok)
	require.Equal(t, "foobar", user.ID)
}

func TestSetSource(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	db = history.SetSource(db, history.Source{ID: "123"})
	source, ok := history.GetSource(db)
	require.True(t, ok)
	require.Equal(t, "123", source.ID)

	db = db.Session(&gorm.Session{})
	source, ok = history.GetSource(db)
	require.True(t, ok)
	require.Equal(t, "123", source.ID)

	db = db.WithContext(context.Background())
	source, ok = history.GetSource(db)
	require.True(t, ok)
	require.Equal(t, "123", source.ID)
}
