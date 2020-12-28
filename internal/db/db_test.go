package db

import (
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"gitlab.com/ranfdev/discepto/internal/models"
)

func mockUser() *models.User {
	return &models.User{
		Name:  "Pippo",
		Email: "pippo@strana.com",
	}

}
func mockUrl() *url.URL {
	url, _ := url.Parse("https://example.com")
	return url
}
func mockEssay(userID int) *models.Essay {
	return &models.Essay{
		Thesis: "Banana is the best fruit",
		Content: `Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...`,
		AttributedToID: userID, // it's a reference, can't mock this
		Tags:           []string{"banana", "fruit", "best"},
		Sources:        []*url.URL{mockUrl()},
		Published:      time.Now(),
	}
}

func init() {
	err := os.Chdir("./../..")
	if err != nil {
		panic(err)
	}
	err = Connect()
	if err != nil {
		panic(err)
	}
	// Reset database before testing
	err = MigrateDown()
	if err != nil {
		panic(err)
	}
	err = MigrateUp()
	if err != nil {
		panic(err)
	}
}
func TestListUsers(t *testing.T) {
	_, err := ListUsers()
	if err != nil {
		t.Error(err)
	}
}
func TestUser(t *testing.T) {
	user := mockUser()
	err := CreateUser(user)
	if err != nil {
		t.Error("Creating user:", err)
	}
	err = DeleteUser(user.ID)
	if err != nil {
		t.Error("Deleting user:", err)
	}
	user.Email = "asdfhasdfkhlkjh"
	err = CreateUser(user)
	// This SHOULD fail
	if err == nil {
		t.Error("Creating user with bad email:", err)
	}
}
func TestEssay(t *testing.T) {
	user := mockUser()
	err := CreateUser(user)
	if err != nil {
		t.Error(err)
		return
	}
	essay := mockEssay(user.ID)
	err = CreateEssay(essay)
	if err != nil {
		t.Errorf("Failed to CreateEssay: %v", err)
	}
	err = DeleteEssay(essay.ID)
	if err != nil {
		t.Errorf("Failed to DeleteEssay: %v", err)
	}
	DeleteUser(user.ID)
}
