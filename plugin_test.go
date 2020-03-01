package history_test

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/suite"
	"os"
	"sort"
	"testing"

	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/stretchr/testify/assert"
	history "github.com/vcraescu/gorm-history"
)

const dbName = "test.db"

type noLogger struct {
}

type Person struct {
	gorm.Model
	FirstName string
	LastName  string
	Address   *string
}

type PersonHistory struct {
	gorm.Model
	history.Entry

	FirstName string
	LastName  string
	Address   *string
}

type PluginTestSuite struct {
	suite.Suite
	db     *gorm.DB
	plugin history.Plugin
}

func (l noLogger) Print(_ ...interface{}) {
}

func (suite *PluginTestSuite) SetupTest() {
	db, err := gorm.Open("sqlite3", dbName)
	if err != nil {
		panic(err)
	}

	suite.db = db
	suite.db.SetLogger(noLogger{})
	suite.plugin = history.Register(suite.db, history.WithVersionFunc(history.TimedVersionFunc))
	err = suite.db.AutoMigrate(Person{}, PersonHistory{}).Error
	if err != nil {
		panic(err)
	}
}

func (suite *PluginTestSuite) TearDownTest() {
	err := os.Remove(dbName)
	if err != nil {
		panic(err)
	}
}

type versions []history.Version

func (v versions) Len() int {
	return len(v)
}

func (v versions) Less(i, j int) bool {
	return v[i] < v[j]
}

func (v versions) Swap(i, j int) {
	s := v[i]
	v[i] = v[j]
	v[j] = s
}

func (Person) CreateHistory() interface{} {
	return PersonHistory{}
}

func TestCopyFn(t *testing.T) {
	address := "address"

	p := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address:   &address,
	}

	h := PersonHistory{}
	assert.NoError(t, history.DefaultCopyFunc(p, &h))

	assert.Equal(t, p.FirstName, h.FirstName)
	assert.Equal(t, p.LastName, h.LastName)
	assert.NotNil(t, h.Address)
	assert.Equal(t, *p.Address, *h.Address)
}

func TestTimedVersionFn(t *testing.T) {
	v, err := history.TimedVersionFunc(history.Context{Action: history.ActionCreate})
	assert.NoError(t, err)
	assert.Zero(t, v)

	n := 100000
	exists := make(map[history.Version]bool)
	vs := make(versions, n)
	for i := 0; i < n; i++ {
		version, err := history.TimedVersionFunc(history.Context{Action: history.ActionUpdate})
		assert.NoError(t, err)
		if len(exists) > 0 {
			_, ok := exists[version]
			if !assert.False(t, ok, fmt.Sprintf("duplicate detected: %d (%d)", version, i)) {
				break
			}

			vs = append(vs, version)
		}

		exists[version] = true
	}

	assert.True(t, sort.IsSorted(vs))
}

func (suite *PluginTestSuite) TestTimedVersionFn() {
	suite.Zero(history.IncrementedVersionFunc(history.Context{Action: history.ActionCreate}))

	p := Person{
		FirstName: "First Name 0",
		LastName:  "Last Name 0",
	}

	suite.db.Save(&p)
	n := 10
	for i := 0; i < n; i++ {
		p.FirstName = fmt.Sprintf("First Name %d", i)
		p.LastName = fmt.Sprintf("First Name %d", i)
		suite.db.Save(&p)
	}

	entries := make([]PersonHistory, n)
	err := suite.
		db.
		Model(PersonHistory{}).
		Order("version asc").
		Find(&entries, "object_id = ?", p.ID).
		Error
	if err != nil {
		panic(err)
	}

	for i, entry := range entries {
		if i == 0 {
			if !suite.Zero(entry.Version) {
				return
			}

			continue
		}

		if !suite.NotZero(entry.Version) {
			return
		}
	}
}

func (suite *PluginTestSuite) TestIncrementedVersionFn() {
	suite.plugin.VersionFunc = history.IncrementedVersionFunc
	suite.plugin.Register()

	suite.Zero(history.IncrementedVersionFunc(history.Context{Action: history.ActionCreate}))

	p := Person{
		FirstName: "First Name 0",
		LastName:  "Last Name 0",
	}

	suite.db.Save(&p)
	n := 10
	for i := 0; i < n; i++ {
		p.FirstName = fmt.Sprintf("First Name %d", i)
		p.LastName = fmt.Sprintf("Last Name %d", i)
		suite.db.Save(&p)
	}

	entries := make([]PersonHistory, n)
	err := suite.
		db.
		Model(PersonHistory{}).
		Order("version asc").
		Find(&entries).
		Error
	if err != nil {
		panic(err)
	}

	for i, entry := range entries {
		if i == 0 {
			if !suite.Zero(entry.Version) {
				return
			}

			continue
		}

		if !suite.NotZero(entry.Version) {
			return
		}
	}
}

func TestPluginTestSuite(t *testing.T) {
	suite.Run(t, new(PluginTestSuite))
}
