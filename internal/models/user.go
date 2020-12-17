package models

type User struct {
	ID int
	Name string
	Email string
	Role Role
	Following []User `gorm:"many2many:user_following"`
}
