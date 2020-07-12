# gorm history [![Go Report Card](https://goreportcard.com/badge/github.com/vcraescu/gorm-history)](https://goreportcard.com/report/github.com/vcraescu/gorm-history) [![Build Status](https://travis-ci.com/vcraescu/gorm-history.svg?branch=master)](https://travis-ci.com/vcraescu/gorm-history) [![Coverage Status](https://coveralls.io/repos/github/vcraescu/gorm-history/badge.svg?branch=master)](https://coveralls.io/github/vcraescu/gorm-history?branch=master) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
History is used to keep a history record of your GORM models.
Each model must be associated with an history model which is a copy of the modified record.

## Install
```
go get github.com/vcraescu/gorm-history
```

## Usage
1. Register the plugin using `history.Register(db)`:

```go
plugin, err := Register(db) // db is a *gorm.DB
if err != nil {
    panic(err)
}
```

2. Define your model and history model:
```go
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
```

3. Your model must implement `history.Recordable` interface:
```go
func (Person) CreateHistory() interface{} {
	return PersonHistory{}
}
```

4. Changes after calling Create, Save, Update will be tracked.


*Note*: By default, the plugin will use time based versioning function for your history record which might not be 100%
accurate in some cases but it is faster because it doesn't require to query the db history table at all.

## Configuration

### Versioning 

* `history.TimedVersionFunc` - It returns the nanonseconds value when the create/update happens;
* `history.IncrementedVersionFunc` - It returns the maximum + 1 of current version column. If you use this versioning 
method than you must tag your history model fields (or embed `history.Entry` struct) like this instead of 
implementing the `history.History` interface:

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
plugin, err := Register(db, history.WithVersionFunc(history.IncrementedVersionFunc)) // db is a *gorm.DB
if err != nil {
    panic(err)
}
```

### Copying 

* `history.DefaultCopyFunc` - copies all the values of the recordable model to history model.

You can change the copy function when you register the plugin if you defined your own function:

```go
func myCopyFunc(r Recordable, history interface{}) error {
    // ...
}

//...

plugin, err := Register(db, history.WithCopyFunc(myCopyFunc)) // db is a *gorm.DB
if err != nil {
    panic(err)
}
```

## License

gorm-history is licensed under the [MIT License](LICENSE).
