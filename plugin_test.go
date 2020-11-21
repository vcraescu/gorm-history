package history

import (
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
	"testing"

	"gorm.io/driver/sqlite"
)

type (
	Person struct {
		gorm.Model

		FirstName       string
		LastName        string
		AddressID       *uint
		Address         *Address
		somePrivateProp string
	}

	PersonHistory struct {
		gorm.Model
		Entry

		FirstName string
		LastName  string
		AddressID uint
	}

	Address struct {
		gorm.Model

		Line1 string
		Line2 string
		City  string
	}

	AddressHistory struct {
		gorm.Model
		Entry

		Line1 string
		Line2 string
		City  string
	}

	PluginTestSuite struct {
		suite.Suite
		db *gorm.DB
	}

	versions []Version
)

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

func (Person) CreateHistory() History {
	return &PersonHistory{}
}

func (Address) CreateHistory() History {
	return &AddressHistory{}
}

func ExamplePlugin() {
	type Person struct {
		gorm.Model

		FirstName string
		LastName  string
	}

	type PersonHistory struct {
		gorm.Model
		Entry
	}

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	db = db.Session(&gorm.Session{})

	err = db.AutoMigrate(Person{}, PersonHistory{})
	if err != nil {
		panic(err)
	}

	plugin := New()
	if err := db.Use(plugin); err != nil {
		return
	}

	p := Person{
		FirstName: "John",
		LastName:  "Doe",
	}
	if err := db.Save(&p).Error; err != nil {
		panic(err)
	}
}

func TestCopyFn(t *testing.T) {
	p := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address: &Address{
			Line1: "line1",
			Line2: "line2",
			City:  "line3",
		},
	}

	h := PersonHistory{}
	require.NoError(t, DefaultCopyFunc(p, &h))
	require.Equal(t, p.FirstName, h.FirstName)
	require.Equal(t, p.LastName, h.LastName)
}

func TestULIDVersion_Version(t *testing.T) {
	version := NewULIDVersion()
	v, err := version.Version(&Context{action: ActionCreate})
	require.NoError(t, err)
	require.Zero(t, v)

	n := 100000
	exists := make(map[Version]bool)
	vs := make(versions, n)
	for i := 0; i < n; i++ {
		version, err := version.Version(&Context{action: ActionUpdate})
		require.NoError(t, err)

		require.NotContains(t, version, exists)

		vs = append(vs, version)

		exists[version] = true
	}

	require.True(t, sort.IsSorted(vs))
}

func (suite *PluginTestSuite) SetupTest() {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	suite.db = db.Session(&gorm.Session{})

	err = suite.db.AutoMigrate(Person{}, PersonHistory{}, Address{}, AddressHistory{})
	if err != nil {
		panic(err)
	}
}

func (suite *PluginTestSuite) TearDownTest() {
	db := suite.db.Session(&gorm.Session{AllowGlobalUpdate: true})
	db.Delete(&Person{})
	db.Delete(&PersonHistory{})
	db.Delete(&Address{})
	db.Delete(&AddressHistory{})
}

func (suite *PluginTestSuite) TestDefaultVersionFunc() {
	plugin := New()
	err := suite.db.Use(plugin)
	suite.Require().NoError(err)

	p := Person{
		FirstName: "First Name 0",
		LastName:  "Last Name 0",
		Address: &Address{
			Line1: "line1",
			Line2: "line2",
			City:  "UK",
		},
	}

	err = suite.db.Save(&p).Error
	suite.Require().NoError(err)

	n := 10
	people := make([]Person, n+1)

	np := Person{}
	suite.Require().NoError(copier.Copy(&np, p))
	people[0] = np

	for i := 1; i <= n; i++ {
		p.FirstName = fmt.Sprintf("First Name %d", i)
		p.LastName = fmt.Sprintf("Last Name %d", i)
		suite.Require().NoError(suite.db.Save(&p).Error)

		np := Person{}
		suite.Require().NoError(copier.Copy(&np, p))

		people[i] = np
	}

	entries := make([]PersonHistory, n)
	err = suite.
		db.
		Model(PersonHistory{}).
		Order("version asc").
		Find(&entries, "object_id = ?", p.ID).
		Error
	suite.Require().NoError(err)
	suite.Require().Len(entries, n+1)

	for i, entry := range entries {
		suite.Equal(fmt.Sprintf("%v", people[i].FirstName), entry.FirstName)
		suite.Equal(fmt.Sprintf("%v", people[i].LastName), entry.LastName)
		suite.Equal(fmt.Sprintf("%v", people[i].ID), entry.ObjectID)

		if i == 0 {
			suite.Require().Zero(entry.Version)

			continue
		}

		suite.Require().NotZero(entry.Version)
	}
}

func (suite *PluginTestSuite) TestDefaultSourceGeneratorFunc() {
	plugin := New()
	err := suite.db.Use(plugin)
	suite.Require().NoError(err)

	p := Person{
		FirstName: "First Name 0",
		LastName:  "Last Name 0",
		Address: &Address{
			Line1: "line1",
			Line2: "line2",
			City:  "UK",
		},
	}

	db := SetSource(suite.db, Source{ID: "123"})
	err = db.Save(&p).Error
	suite.Require().NoError(err)

	var ph PersonHistory
	err = suite.
		db.
		Find(&ph, "object_id = ?", p.ID).
		Error
	suite.Require().NoError(err)

	var sourceID string
	suite.Require().NotEmpty(ph.SourceID)
	if sourceID == "" {
		sourceID = ph.SourceID
	}

	suite.Require().Equal(ph.SourceID, sourceID)

	var ah AddressHistory
	err = suite.
		db.
		Find(&ah, "object_id = ?", p.AddressID).
		Error
	suite.Require().NoError(err)
	suite.Require().Equal(sourceID, ah.SourceID)
}

func (suite *PluginTestSuite) TestUpdateColumns() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	p := Person{
		FirstName: "First Name 0",
		LastName:  "Last Name 0",
	}

	err := suite.db.Save(&p).Error
	suite.Require().NoError(err)

	n := 10
	people := make([]Person, n+1)

	np := Person{}
	suite.Require().NoError(copier.Copy(&np, p))
	people[0] = np

	for i := 1; i <= n; i++ {
		err := suite.db.
			Omit(clause.Associations).
			Model(&p).
			Updates(
				Person{
					FirstName: fmt.Sprintf("First Name %d", i),
					LastName:  fmt.Sprintf("First Name %d", i),
				},
			).Error
		suite.Require().NoError(err)

		np := Person{}
		suite.Require().NoError(copier.Copy(&np, p))

		people[i] = np
	}

	entries := make([]PersonHistory, n)
	err = suite.
		db.
		Model(PersonHistory{}).
		Order("version asc").
		Find(&entries, "object_id = ?", p.ID).
		Error
	suite.Require().NoError(err)
	suite.Require().Len(entries, n+1)

	for i, entry := range entries {
		suite.Equal(fmt.Sprintf("%v", people[i].FirstName), entry.FirstName)
		suite.Equal(fmt.Sprintf("%v", people[i].LastName), entry.LastName)
		suite.Equal(fmt.Sprintf("%v", people[i].ID), entry.ObjectID)

		if i == 0 {
			suite.Require().Zero(entry.Version)

			continue
		}

		suite.Require().NotZero(entry.Version)
	}
}

func (suite *PluginTestSuite) TestBatchInsert() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	n := 10
	people := make([]Person, n)
	for i, _ := range people {
		people[i] = Person{
			FirstName: fmt.Sprintf("First Name %d", i),
			LastName:  fmt.Sprintf("Last Name %d", i),
			Address: &Address{
				Line1: fmt.Sprintf("Line %d", i),
			},
		}
	}

	err := suite.db.Create(&people).Error
	suite.Require().NoError(err)

	entries := make([]PersonHistory, n)
	err = suite.
		db.
		Order("version asc").
		Find(&entries).
		Error
	suite.Require().NoError(err)
	suite.Require().Len(entries, n)

	for i, entry := range entries {
		suite.Equal(fmt.Sprintf("%v", people[i].FirstName), entry.FirstName)
		suite.Equal(fmt.Sprintf("%v", people[i].LastName), entry.LastName)
		suite.Equal(fmt.Sprintf("%v", people[i].ID), entry.ObjectID)
		suite.Equal(ActionCreate, entry.Action)
		suite.Zero(entry.Version)
	}
}

func (suite *PluginTestSuite) TestBatchUpdates() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	n := 10
	people := make([]Person, n)
	for i, _ := range people {
		people[i] = Person{
			FirstName: fmt.Sprintf("First Name %d", i),
			LastName:  fmt.Sprintf("Last Name %d", i),
			Address: &Address{
				Line1: fmt.Sprintf("Line %d", i),
			},
		}
	}

	err := suite.db.Create(people).Error
	suite.Require().NoError(err)

	err = suite.db.
		Session(&gorm.Session{AllowGlobalUpdate: true}).
		Updates(Person{FirstName: "foobar"}).
		Error
	suite.Require().Error(err)
	suite.Error(ErrUnsupportedOperation, err)
}

func (suite *PluginTestSuite) TestUserAndSource() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	user := User{
		ID:    "123",
		Email: "john@doe.com",
	}
	p := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address: &Address{
			Line1: "Line 1",
			Line2: "Line 2",
			City:  "Iasi",
		},
	}

	sourceID := "mySourceID"
	db := SetUser(suite.db, user)
	db = SetSource(db, Source{ID: sourceID})
	err := db.Create(&p).Error
	suite.Require().NoError(err)

	p.Address = nil
	p.FirstName = "Jane"
	err = db.Save(&p).Error
	suite.Require().NoError(err)

	var count int64
	err = suite.
		db.
		Model(&PersonHistory{}).
		Where("object_id = ? AND user_id = ?", p.ID, user.ID).
		Count(&count).
		Error
	suite.Require().NoError(err)
	suite.Require().EqualValues(2, count)

	count = 0
	err = suite.
		db.
		Model(&AddressHistory{}).
		Where("object_id = ? AND user_id = ?", p.AddressID, user.ID).
		Count(&count).
		Error
	suite.Require().NoError(err)
	suite.Require().EqualValues(1, count)

	p = Person{
		FirstName: "John",
		LastName:  "Doe",
	}

	err = suite.db.Create(&p).Error
	suite.Require().NoError(err)

	err = suite.db.
		Model(&PersonHistory{}).
		Where("object_id = ? and user_email <> ?", p.ID, user.ID).
		Count(&count).
		Error
	suite.Require().NoError(err)
	suite.Require().EqualValues(1, count)
}

func (suite *PluginTestSuite) TestDisabled() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	p := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address: &Address{
			Line1: "Line 1",
			Line2: "Line 2",
			City:  "Iasi",
		},
	}

	db := Disable(suite.db)
	err := db.Create(&p).Error
	suite.Require().NoError(err)

	p.FirstName = "Jane"
	err = db.Save(&p).Error
	suite.Require().NoError(err)

	var count int64
	err = suite.db.Model(&PersonHistory{}).Where("object_id = ?", p.ID).Count(&count).Error
	suite.Require().NoError(err)
	suite.Empty(count)

	err = suite.db.Model(&AddressHistory{}).Where("object_id = ?", p.AddressID).Count(&count).Error
	suite.Require().NoError(err)
	suite.Empty(count)
}

func (suite *PluginTestSuite) TestSave() {
	plugin := New()
	if err := suite.db.Use(plugin); err != nil {
		panic(err)
	}

	p := Person{
		FirstName: "John",
		LastName:  "Doe",
		Address: &Address{
			Line1: "Line 1",
			Line2: "Line 2",
			City:  "Iasi",
		},
	}

	err := suite.db.Save(&p).Error
	suite.Require().NoError(err)

	p.FirstName = "Jane"
	err = suite.db.Save(&p).Error
	suite.Require().NoError(err)

	var count int64
	err = suite.
		db.
		Model(PersonHistory{}).
		Where("object_id = ?", p.ID).
		Count(&count).
		Error
	suite.Require().NoError(err)
	suite.EqualValues(2, count)
}

func TestPluginTestSuite(t *testing.T) {
	suite.Run(t, new(PluginTestSuite))
}
