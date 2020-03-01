package history_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	history "github.com/vcraescu/gorm-history"
)

type UtilModel struct {
	Name string
}

type UtilModelHistory struct {
	history.Entry
	Name string
}

type UtilModel2History struct {
	Name     string
	Version  *history.Version `gorm-history:"version"`
	ObjectID *string          `gorm-history:"objectID"`
}

type UtilModel3History struct {
	Name     string
	Version  history.Version `gorm-history:"version"`
	ObjectID string          `gorm-history:"objectID"`
	Action   history.Action  `gorm-history:"action"`
}

func TestIsHistory(t *testing.T) {
	assert.False(t, history.IsHistory(UtilModel{}))
	assert.True(t, history.IsHistory(UtilModelHistory{}))
}

func TestGetFieldsByTag(t *testing.T) {
	fields := history.GetFieldsByTag(UtilModelHistory{}, history.Tag)
	if !assert.Len(t, fields, 3) {
		return
	}
	assert.ElementsMatch(
		t,
		[]string{"Version", "ObjectID", "Action"},
		[]string{fields[0].Name, fields[1].Name, fields[2].Name},
	)

	fields = history.GetFieldsByTag(UtilModel{}, history.Tag)
	assert.Empty(t, fields)
}

func TestGetHistoryFields(t *testing.T) {
	fields := history.GetHistoryFields(UtilModelHistory{})
	if !assert.Len(t, fields, 3) {
		return
	}
	assert.ElementsMatch(
		t,
		[]string{"Version", "ObjectID", "Action"},
		[]string{fields[0].Name, fields[1].Name, fields[2].Name},
	)

	fields = history.GetHistoryFields(UtilModel2History{})
	if !assert.Len(t, fields, 2) {
		return
	}
	assert.ElementsMatch(
		t,
		[]string{"Version", "ObjectID"},
		[]string{fields[0].Name, fields[1].Name},
	)

	fields = history.GetHistoryFields(UtilModel3History{})
	if !assert.Len(t, fields, 3) {
		return
	}
	assert.ElementsMatch(
		t,
		[]string{"Version", "ObjectID", "Action"},
		[]string{fields[0].Name, fields[1].Name, fields[2].Name},
	)

	fields = history.GetHistoryFields(UtilModel{})
	assert.Empty(t, fields)
}

func TestSetHistoryFields(t *testing.T) {
	h := UtilModelHistory{}
	if !assert.NoError(t, history.SetHistoryFields(&h, uint(100), 200, history.ActionCreate)) {
		return
	}

	assert.Equal(t, uint(100), h.ObjectID)
	assert.Equal(t, history.Version(200), h.Version)
	assert.Equal(t, history.ActionCreate, h.Action)

	h2 := UtilModel2History{}
	if !assert.NoError(t, history.SetHistoryFields(&h2, "100", 200, history.ActionCreate)) {
		return
	}
	assert.NotNil(t, h2.ObjectID)
	assert.Equal(t, "100", *h2.ObjectID)
	assert.NotNil(t, h2.Version)
	assert.Equal(t, history.Version(200), *h2.Version)

	h3 := UtilModel3History{}
	assert.NoError(t, history.SetHistoryFields(&h3, "100", 200, history.ActionCreate))
	assert.Equal(t, "100", h3.ObjectID)
	assert.NotNil(t, h3.Version)
	assert.Equal(t, history.Version(200), h3.Version)
}

func TestSetStructFieldByName(t *testing.T) {
	h := UtilModelHistory{}

	assert.NoError(t, history.SetStructFieldByName(&h, "ObjectID", uint(10)))
	assert.Equal(t, uint(10), h.ObjectID)
	assert.IsType(
		t,
		history.FieldNotFoundError{},
		history.SetStructFieldByName(&h, "ObjectI", "test"),
	)
	assert.Error(t, history.SetStructFieldByName(h, "ObjectID", "test"))
}

func TestSetStructFieldByNameWithPtrs(t *testing.T) {
	h := UtilModel2History{}

	assert.NoError(t, history.SetStructFieldByName(&h, "ObjectID", "test"))
	assert.Equal(t, "test", *h.ObjectID)

	assert.IsType(
		t,
		history.FieldNotFoundError{},
		history.SetStructFieldByName(&h, "ObjectI", "test"),
	)
	assert.Error(t, history.SetStructFieldByName(h, "ObjectID", "test"))
}
