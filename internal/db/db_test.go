package db

import (
	"gitlab.com/ranfdev/discepto/internal/models"
	"os"
	"testing"
)

func init() {
	err := os.Chdir("./../..")
	if err != nil {
		panic(err)
	}
	err = Connect()
	if err != nil {
		panic(err)
	}
	err = Migrate()
	if err != nil {
		panic(err)
	}
}
func TestCreateUser(t *testing.T) {
	user := models.User{
		Name:   "Pippo",
		Email:  "pippo@strana.com",
		RoleID: models.RoleAdmin,
	}
	err := CreateUser(&user)
	if err != nil {
		t.Error(err)
	}
}
func TestCreateUserBadEmail(t *testing.T) {
	user := models.User{
		Name:   "Pippo",
		Email:  "pippoasdfjhasdflkjhs",
		RoleID: models.RoleAdmin,
	}
	err := CreateUser(&user)
	// The email is invalid, so there should be an error
	if err == nil {
		t.Error(err)
	}
}
func TestListUsers(t *testing.T) {
	_, err := ListUsers()
	if err != nil {
		t.Error(err)
	}
}
