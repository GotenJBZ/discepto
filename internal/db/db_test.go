package db

import (
	"net/url"
	"os"
	"testing"
	"time"

	"gitlab.com/ranfdev/discepto/internal/models"
)

const mockPasswd = "123456789" // hackerman
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
		PostedIn:       "mock",
	}
}
func mockSubdiscepto() *models.Subdiscepto {
	return &models.Subdiscepto{
		Name:        "mock",
		Description: "Mock subdiscepto",
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
	users, err := ListUsers()
	if err != nil {
		t.Fatalf("ListUsers() = %v, %v, want users, nil", users, err)
	}
}
func TestUser(t *testing.T) {
	user := mockUser()
	passwd := mockPasswd
	err := CreateUser(user, passwd)
	if err != nil {
		t.Fatalf(
			"CreateUser(%v, %v) = %v, want nil",
			user,
			passwd,
			err,
		)
	}
	// With bad email
	user.Email = "asdfhasdfkhlkjh"
	err = CreateUser(user, passwd)
	// This SHOULD fail
	if err == nil {
		t.Fatalf(
			"CreateUser(%v, %v) = %v, want error",
			user,
			passwd,
			err,
		)
	}

	// Clean
	DeleteUser(user.ID)
}
func TestAuth(t *testing.T) {
	user := mockUser()
	passwd := mockPasswd
	_ = CreateUser(user, passwd)

	// With a bad passwd
	passwd = "93sdjfhkasdhfkjha"
	token, err := Login(user.Email, passwd)
	if err == nil {
		t.Fatalf("Login(%v, %v) = %v, %v, want \"\", err", user, passwd, token, err)
	}

	// Normal login
	passwd = mockPasswd
	token, err = Login(user.Email, passwd)
	if err != nil {
		t.Fatalf("Login(%v, %v) = %v, %v, want token, nil", user, passwd, token, err)
	}

	// Retrieve user by token
	user2, err := GetUserByToken(token)
	if err != nil {
		t.Fatalf("GetUserByToken(%v) = %v, %v, want user, nil", token, user2, err)
	}
	if user.ID != user2.ID {
		t.Fatalf("User IDs are different: %v, %v", user, user2)
	}

	// Sign out
	err = Signout(token)
	if err != nil {
		t.Fatalf("Signout(%v) = %v, want nil", token, err)
	}
	DeleteUser(user.ID)
}
func TestRole(t *testing.T) {
	user := mockUser()
	_ = CreateUser(user, mockPasswd)
	role, err := GetGlobalRole(user.ID)
	if err != nil {
		t.Fatalf("GetGlobalRole(%v) = %v, %v, want role, nil", user.ID, role, err)
	}
	DeleteUser(user.ID)
}
func TestEssay(t *testing.T) {
	user := mockUser()
	CreateUser(user, mockPasswd)
	err := CreateSubdiscepto(mockSubdiscepto(), user.ID)
	if err != nil {
		t.Fatalf("CreateSubdiscepto(%v, %v) = %v, want nil", mockSubdiscepto(), user.ID, err)
	}

	essay := mockEssay(user.ID)
	err = CreateEssay(essay)
	if err != nil {
		t.Fatalf("CreateEssay(%v) = %v, want nil", essay, err)
	}

	essays, err := ListEssays("mock")
	if err != nil {
		t.Fatalf("ListEssays(%v) = %v, %v want essays, nil", "mock", essays, err)
	}

	err = DeleteEssay(essay.ID)
	if err != nil {
		t.Fatalf("DeleteEssay(%v) = %v, want nil", essay.ID, err)
	}
	DeleteUser(user.ID)
}
func TestVotes(t *testing.T) {
	// Setup needed data
	user := mockUser()
	_ = CreateUser(user, mockPasswd)
	essay := mockEssay(user.ID)
	_ = CreateEssay(essay)

	// Actual test
	count, err := CountVotes(essay.ID, models.VoteTypeUpvote)
	if err != nil || count != 0 {
		t.Fatalf("CountVotes(%v, %v) = %v, %v, want 0, nil",
			essay.ID, models.VoteTypeUpvote, count, err)
	}

	// Add upvote
	vote := &models.Vote{
		UserID:   user.ID,
		EssayID:  essay.ID,
		VoteType: models.VoteTypeUpvote,
	}
	err = CreateVote(vote)
	if err != nil {
		t.Fatalf("CreateVote(%v) = %v, want nil", vote, err)
	}
	count, err = CountVotes(essay.ID, models.VoteTypeUpvote)
	if err != nil || count != 1 {
		t.Fatalf("CountVotes(%v, %v) = %v, %v, want 1, nil",
			essay.ID, models.VoteTypeUpvote, count, err)
		return
	}

	// Delete (needed to change vote type for same user)
	err = DeleteVote(vote.EssayID, vote.UserID)
	if err != nil {
		t.Fatalf("DeleteVote(%v, %v) = %v, want nil",
			vote.EssayID, vote.UserID, err)
	}

	// Create downvote
	vote = &models.Vote{
		UserID:   user.ID,
		EssayID:  essay.ID,
		VoteType: models.VoteTypeDownvote,
	}
	err = CreateVote(vote)
	if err != nil {
		t.Fatalf("CreateVote(%v) = %v, want nil", vote, err)
	}
	count, err = CountVotes(essay.ID, models.VoteTypeDownvote)
	if err != nil || count != 1 {
		t.Fatalf("CountVotes(%v, %v) = %v, %v, want 1, nil",
			essay.ID, models.VoteTypeDownvote, count, err)
	}

	// Clean
	err = DeleteVote(vote.EssayID, vote.UserID)
	if err != nil {
		t.Fatalf("DeleteVote(%v, %v) = %v, want nil",
			vote.EssayID, vote.UserID, err)
	}

	DeleteEssay(essay.ID)
	DeleteUser(user.ID)
}
func TestSubdiscepto(t *testing.T) {
	// Setup needed data
	user := mockUser()
	CreateUser(user, mockPasswd)

	// Actual test
	subdis := &models.Subdiscepto{
		Name:        "subtest",
		Description: "here we talk about tests",
	}
	err := CreateSubdiscepto(subdis, user.ID)
	if err != nil {
		t.Fatalf("CreateSubdiscepto(%v, %v) = %v, want nil", subdis, user.ID, err)
	}

	err = DeleteSubdiscepto(subdis.Name)
	if err != nil {
		t.Fatalf("DeleteSubdiscepto(%v) = %v, want nil", subdis.Name, err)
	}

	// Clean
	DeleteUser(user.ID)
}
