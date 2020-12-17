package models

type Subdiscepto struct {
	Name string
	Description string
	Topic string
	Members []User `gorm:"many2many:subdiscepto_members"`
}
