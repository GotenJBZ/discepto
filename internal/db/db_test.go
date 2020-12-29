package db

import (
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
	users, err := ListUsers()
	if err != nil {
		t.Fatalf("ListUsers() = %v, %v, want users, nil", users, err)
	}
}
func TestUser(t *testing.T) {
	user := mockUser()
	err := CreateUser(user)
	if err != nil {
		t.Fatalf("CreateUser(%v) = %v, want nil", user, err)
	}
	err = DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser(%d) = %v, want nil", user.ID, err)
	}
	user.Email = "asdfhasdfkhlkjh"
	err = CreateUser(user)
	// This SHOULD fail
	if err == nil {
		t.Fatalf("CreateUser(%v) = %v, want error", user, err)
	}

	// Clean
	DeleteUser(user.ID)
}
func TestEssay(t *testing.T) {
	user := mockUser()
	err := CreateUser(user)
	if err != nil {
		t.Fatal(err)
	}
	essay := mockEssay(user.ID)
	err = CreateEssay(essay)
	if err != nil {
		t.Fatalf("CreateEssay(%v) = %v, want nil", essay, err)
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
	_ = CreateUser(user)
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
	CreateUser(user)

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
