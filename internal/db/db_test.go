package db

import (
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"gitlab.com/ranfdev/discepto/internal/models"
)

const mockPasswd = "123456789" // hackerman
const mockSubName = "mock"
const mockSubName2 = "mock2"

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
		PostedIn:       mockSubName,
		ReplyType:      models.ParseReplyType(""),
	}
}
func mockSubdiscepto() *models.Subdiscepto {
	return &models.Subdiscepto{
		Name:        mockSubName,
		Description: "Mock subdiscepto",
	}
}
func mockSubdiscepto2() *models.Subdiscepto {
	return &models.Subdiscepto{
		Name:        mockSubName2,
		Description: "Mock subdiscepto 2",
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

	essays, err := ListEssays(mockSubName)
	if err != nil {
		t.Fatalf("ListEssays(%v) = %v, %v want essays, nil", mockSubName, essays, err)
	}

	// Test list recent essays from joined subs
	// Create and fill second sub
	CreateSubdiscepto(mockSubdiscepto2(), user.ID)
	essay2 := mockEssay(user.ID)
	CreateEssay(essay2)

	// list
	subs := []string{mockSubName, mockSubName2}
	essays, err = ListRecentEssaysIn(subs)
	if err != nil || len(essays) < 2 {
		t.Fatalf("ListRecentEssaysIn(%v) = %v,%v want essays (len > 2), nil", subs, essays, err)
	}

	// Test list essays in favor
	// Add initial votes
	essay3 := mockEssay(user.ID)
	essay3.InReplyTo = sql.NullInt32{Int32: int32(essay2.ID), Valid: true}
	essay3.ReplyType = models.ReplyTypeInFavor
	CreateEssay(essay3)
	// list
	essays, err = ListEssaysInFavor(essay2.ID)
	if err != nil || len(essays) != 1 {
		t.Fatalf("ListEssaysInFavor(%v) = %v,%v want essays, nil", essay2.ID, essays, err)
	}

	// Clean
	toDelete := []*models.Essay{
		essay3,
		essay2,
		essay,
	}
	for _, es := range toDelete {
		err = DeleteEssay(es.ID)
		if err != nil {
			t.Fatalf("DeleteEssay(%v) = %v, want nil", es.ID, err)
		}
		DeleteSubdiscepto(es.PostedIn)
	}
	DeleteUser(user.ID)
}
func TestVotes(t *testing.T) {
	// Setup needed data
	user := mockUser()
	_ = CreateUser(user, mockPasswd)
	essay := mockEssay(user.ID)
	CreateSubdiscepto(mockSubdiscepto(), user.ID)
	_ = CreateEssay(essay)

	// Actual test
	upvotes, downvotes, err := CountVotes(essay.ID)
	if err != nil || upvotes != 0 || downvotes != 0 {
		t.Fatalf("CountVotes(%v) = %v,%v,%v, want 0, 0, nil",
			essay.ID, upvotes, downvotes, err)
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

	// Check added upvote
	upvotes, downvotes, err = CountVotes(essay.ID)
	if err != nil || upvotes == 0 || downvotes != 0 {
		t.Fatalf("CountVotes(%v) = %v,%v,%v, want upvotes, 0, nil",
			essay.ID, upvotes, downvotes, err)
	}

	// Delete (needed to change vote type for same user)
	err = DeleteVote(vote.EssayID, vote.UserID)
	if err != nil {
		t.Fatalf("DeleteVote(%v, %v) = %v, want nil",
			vote.EssayID, vote.UserID, err)
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

	subs, err := ListSubdisceptos()
	if err != nil || len(subs) == 0 {
		t.Fatalf("ListSubdisceptos() = %v, %v, want subs (len >= 1), nil", subdis, err)
	}

	// Join a sub
	user2 := mockUser()
	user2.Email += "as"

	CreateUser(user2, mockPasswd)

	err = JoinSubdiscepto(mockSubName, user2.ID)
	if err != nil {
		t.Fatalf("JoinSubdiscepto(%v,%v) = %v, want nil", mockSubName, user2.ID, err)
	}

	mySubs, err := ListMySubdisceptos(user2.ID)
	if mySubs[0] != mockSubName || err != nil {
		t.Fatalf("ListMySubdisceptos(%v) = %v,%v want mySubs, nil", user2.ID, mySubs, err)
	}

	err = LeaveSubdiscepto(mockSubName, user2.ID)
	if err != nil {
		t.Fatalf("LeaveSubdiscepto(%v,%v) = %v, want nil", mockSubName, user2.ID, err)
	}

	err = DeleteSubdiscepto(subdis.Name)
	if err != nil {
		t.Fatalf("DeleteSubdiscepto(%v) = %v, want nil", subdis.Name, err)
	}

	// Clean
	DeleteUser(user.ID)
	DeleteUser(user2.ID)
}
