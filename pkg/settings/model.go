package settings

import "gorm.io/datatypes"

type Model struct {
	Key   string `gorm:"primaryKey;size:255;not null"`
	Value datatypes.JSON
	Order int
}

func (Model) TableName() string {
	return "settings"
}
