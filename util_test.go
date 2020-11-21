package history

import (
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

type (
	ID string
)

func (p ID) IsZero() bool {
	return p == "zero"
}

func TestMakePtr(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
	}{
		{
			name:  "non pointer",
			value: Person{},
		},
		{
			name:  "pointer",
			value: &Person{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := makePtr(test.value)
			v := reflect.ValueOf(p)
			require.Equal(t, reflect.Ptr, v.Kind())
			require.NotEqual(t, reflect.Ptr, reflect.Indirect(v).Kind())
		})
	}
}

func TestUnsetStructField(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		fieldName string
		err       bool
	}{
		{
			name: "valid",
			value: &Person{
				FirstName: "John",
			},
			fieldName: "FirstName",
		},
		{
			name: "non pointer returns an err",
			value: Person{
				FirstName: "John",
			},
			err: true,
		},
		{
			name: "non struct returns an err",
			value: func() *int {
				i := 100
				return &i
			}(),
			err: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := unsetStructField(test.value, test.fieldName)
			if test.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			v := reflect.ValueOf(test.value)
			require.True(t, v.Elem().FieldByName(test.fieldName).IsZero())
		})
	}
}

func TestGetPrimaryKeyValue(t *testing.T) {
	a := require.New(t)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	a.NoError(err)

	db = db.Session(&gorm.Session{})

	type Foo struct {
		Bar string
	}

	type EntityWithIsZeroerPK struct {
		ID   ID
		Name string
	}

	tests := []struct {
		name     string
		object   interface{}
		expected primaryKeyField
		err      bool
	}{
		{
			name: "value object",
			object: Person{
				Model: gorm.Model{
					ID: 444,
				},
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: primaryKeyField{
				name:   "ID",
				value:  uint(444),
				isZero: false,
			},
		},
		{
			name: "pointer object",
			object: &Person{
				Model: gorm.Model{
					ID: 444,
				},
				FirstName: "John",
				LastName:  "Doe",
			},
			expected: primaryKeyField{
				name:   "ID",
				value:  uint(444),
				isZero: false,
			},
		},
		{
			name: "object with no primary key",
			object: Foo{
				Bar: "foobar",
			},
			err: true,
		},
		{
			name: "object with IsZeroer PK",
			object: EntityWithIsZeroerPK{
				ID:   "zero",
				Name: "John",
			},
			expected: primaryKeyField{
				name:   "ID",
				value:  ID("zero"),
				isZero: true,
			},
		},
		{
			name:   "nil object",
			object: nil,
			err:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.object != nil {
				err = db.Statement.Parse(test.object)
				require.NoError(t, err)
			}

			actual, err := getPrimaryKeyValue(db, reflect.ValueOf(test.object))
			if test.err {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, test.expected, *actual)
		})
	}
}
