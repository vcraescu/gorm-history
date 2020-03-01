package history

type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
)

type Recordable interface {
	CreateHistory() interface{}
}

type History interface {
	SetHistoryVersion(version Version)
	SetHistoryObjectID(id interface{})
	SetHistoryAction(action Action)
}

type Entry struct {
	Version  Version `gorm-history:"version"`
	ObjectID uint    `gorm:"index" gorm-history:"objectID"`
	Action   Action  `gorm:"type: string" gorm-history:"action"`
}

func (l *Entry) SetHistoryVersion(version Version) {
	l.Version = version
}

func (l *Entry) SetHistoryObjectID(id interface{}) {
	l.ObjectID = id.(uint)
}

func (l *Entry) SetHistoryAction(action Action) {
	l.Action = action
}
