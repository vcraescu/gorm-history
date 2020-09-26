# gorm history [![Go Report Card](https://goreportcard.com/badge/github.com/vcraescu/gorm-history)](https://goreportcard.com/report/github.com/vcraescu/gorm-history) [![Build Status](https://travis-ci.com/vcraescu/gorm-history.svg?branch=master)](https://travis-ci.com/vcraescu/gorm-history) [![Coverage Status](https://coveralls.io/repos/github/vcraescu/gorm-history/badge.svg?branch=master)](https://coveralls.io/github/vcraescu/gorm-history?branch=master) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
You can use the plugin to keep a history of your GORM models changes. Basically it keeps a ledger of your model state.
Each model must be associated with a history model which will be a copy of the modified record.

## Install
```
go get github.com/vcraescu/gorm-history
```

## Usage

1. Register the plugin using `db.Use(history.New())`:

```go
type Person struct {
    gorm.Model

    FirstName string
    LastName  string
}

type PersonHistory struct {
    gorm.Model
    Entry
}

func main() {
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

    db = SetUser(db, history.User{
        ID:    123,
        Email: "john@doe.com",
    })
    db = SetSource(db, history.Source{
        ID:   "1c059a03-3b14-4017-ae33-5337860ec35f",
        Type: "ampq",
    })

    p := Person{
        FirstName: "John",
        LastName:  "Doe",
    }
    if err := db.Save(&p).Error; err != nil {
        panic(err)
    }
}
```

2. Your model must implement `history.Recordable` interface:
```go
func (Person) CreateHistory() interface{} {
	return PersonHistory{}
}
```

3. Changes after calling Create, Save and Update will be recorded as long as you pass in the original object. 

```go
if err := db.Model(&p).Update("first_name", "Jane").Error; err != nil {
    panic(err)
}
```

## Configuration

### Versioning 

By default, the plugin generates a new ULID for each history record. You can implement your own versioning function.

```go
type PersonHistory struct {
	gorm.Model
	history.Entry
}

// or 
type PersonHistory struct {
	gorm.Model

	Version  Version `gorm-history:"version"`
	ObjectID uint    `gorm:"index" gorm-history:"objectID"`
	Action   Action  `gorm:"type: string" gorm-history:"action"`
}
```


You can change the versioning function when you register the plugin:
```go
if err := db.Use(history.New(history.WithVersionFunc(MyVersionFunc))); err != nil {
    panic(err)
}
```

### Copying 

* `history.DefaultCopyFunc` - copies all the values of the recordable model to history model.

You can change the copy function when you register the plugin if you defined your own copying function:

```go
func myCopyFunc(r Recordable, history interface{}) error {
    // ...
}

//...
if err := db.Use(history.New(history.WithCopyFunc(myCopyFunc))); err != nil {
    panic(err)
}
```

## License

gorm-history is licensed under the [MIT License](LICENSE).
