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
func TestCreateEssay(t *testing.T) {
	myurl, _ := url.Parse("https://fruit.com")
	essay := &models.Essay{
		Thesis: "Banana is the best fruit",
		Content: `Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...`,
		AttributedToID: 1,
		Tags:           []string{"banana", "fruit", "best"},
		Sources:        []*url.URL{myurl},
		Published:      time.Now(),
	}
	err := CreateEssay(essay)
	if err != nil {
		t.Errorf("Failed to create essay: %v", err)
	}
}
