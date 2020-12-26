package db

import (
	"net/url"
	"os"
	"testing"
	"time"

	"gitlab.com/ranfdev/discepto/internal/models"
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
	err = MigrateUp()
	if err != nil {
		panic(err)
	}
}
func TestCreateUser(t *testing.T) {
	user := models.User{
		Name:  "Pippo",
		Email: "pippo@strana.com",
	}
	err := CreateUser(&user)
	if err != nil {
		t.Error(err)
	}
}
func TestCreateUserBadEmail(t *testing.T) {
	user := models.User{
		Name:  "Pippo",
		Email: "pippoasdfjhasdflkjhs",
	}
	err := CreateUser(&user)
	// The email is invalid, so there should be an error
	if err == nil {
		t.Error(err)
	}
}
func TestDeleteUser(t *testing.T) {

}
func TestListUsers(t *testing.T) {
	_, err := ListUsers()
	if err != nil {
		t.Error(err)
	}
}
func TestCreateEssay(t *testing.T) {
	users, err := ListUsers()
	if err != nil {
		t.Error(err)
	}
	myurl, _ := url.Parse("https://fruit.com")
	essay := &models.Essay{
		Thesis: "Banana is the best fruit",
		Content: `Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...`,
		AttributedToID: users[0].ID,
		Tags:           []string{"banana", "fruit", "best"},
		Sources:        []*url.URL{myurl},
		Published:      time.Now(),
	}
	err = CreateEssay(essay)
	if err != nil {
		t.Errorf("Failed to create essay: %v", err)
	}
}
func TestDeleteEssay(t *testing.T) {

}
