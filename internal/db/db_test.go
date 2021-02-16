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
		ReplyType:      models.ReplyTypeGeneral,
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

var db DB

func init() {
	err := os.Chdir("./../..")
	if err != nil {
		panic(err)
	}
	envConfig := models.ReadEnvConfig()

	// Reset database before testing
	err = MigrateDown(envConfig.DatabaseURL)
	if err != nil {
		panic(err)
	}
	err = MigrateUp(envConfig.DatabaseURL)
	if err != nil {
		panic(err)
	}

	db, err = Connect(&envConfig)
	if err != nil {
		panic(err)
	}
}
func TestUser(t *testing.T) {
	user2 := mockUser()
	user2.Email = "asdasdasdfjh"
	testData := []struct{
		user *models.User
		err error
	}{
		{user: mockUser(),err: nil},
		{user: mockUser(),err: ErrEmailAlreadyUsed},
		{user: user2, err: ErrBadEmailSyntax},
	}

	passwd := mockPasswd
	for _, td := range testData {
		err := db.CreateUser(td.user, passwd)
		if err != td.err {
			t.Fatalf("CreateUser(%v, %v) = %v, want %v",
				td.user, passwd, err, td.err)
		}

	}

	users, err := db.ListUsers()
	if len(users) == 0 || err != nil {
		t.Fatalf("ListUsers() = %v,%v, n>0, want nil",
			len(users), err)
	}

	for _, td := range testData {
		if td.err == nil {
			err = db.DeleteUser(td.user.ID)
			if err != nil {
				t.Fatalf("DeleteUser(%v) = %v, want nil",
					td.user.ID, err)
			}
		}
	}
}
func TestAuth(t *testing.T) {
	user := mockUser()
	passwd := mockPasswd
	_ = db.CreateUser(user, passwd)

	// With a bad passwd
	passwd = "93sdjfhkasdhfkjha"
	token, err := db.Login(user.Email, passwd)
	if err == nil {
		t.Fatalf("Login(%v, %v) = %v, %v, want \"\", err", user, passwd, token, err)
	}

	// Normal login
	passwd = mockPasswd
	token, err = db.Login(user.Email, passwd)
	if err != nil {
		t.Fatalf("Login(%v, %v) = %v, %v, want token, nil", user, passwd, token, err)
	}

	// Retrieve user by token
	user2, err := db.GetUserByToken(token)
	if err != nil {
		t.Fatalf("GetUserByToken(%v) = %v, %v, want user, nil", token, user2, err)
	}
	if user.ID != user2.ID {
		t.Fatalf("User IDs are different: %v, %v", user, user2)
	}

	// Sign out
	err = db.Signout(token)
	if err != nil {
		t.Fatalf("Signout(%v) = %v, want nil", token, err)
	}
	db.DeleteUser(user.ID)
}
func TestRoles(t *testing.T) {
	user := mockUser()
	_ = db.CreateUser(user, mockPasswd)
	sub := sql.NullString{Valid: false}
	roles, err := db.GetRoles(user.ID, sub)
	if err != nil {
		t.Fatalf("GetRoles(%v, %v) = %v, %v, want roles, nil", user.ID, sub, roles, err)
	}
	db.DeleteUser(user.ID)
}
func TestEssay(t *testing.T) {
	user := mockUser()
	db.CreateUser(user, mockPasswd)
	err := db.CreateSubdiscepto(mockSubdiscepto(), user.ID)
	if err != nil {
		t.Fatalf("CreateSubdiscepto(%v, %v) = %v, want nil", mockSubdiscepto(), user.ID, err)
	}

	essay := mockEssay(user.ID)
	err = db.CreateEssay(essay)
	if err != nil {
		t.Fatalf("CreateEssay(%v) = %v, want nil", essay, err)
	}

	essays, err := db.ListEssays(mockSubName)
	if err != nil {
		t.Fatalf("ListEssays(%v) = %v, %v want essays, nil", mockSubName, essays, err)
	}

	// Test list recent essays from joined subs
	// Create and fill second sub
	db.CreateSubdiscepto(mockSubdiscepto2(), user.ID)
	essay2 := mockEssay(user.ID)
	db.CreateEssay(essay2)

	// list
	subs := []string{mockSubName, mockSubName2}
	essays, err = db.ListRecentEssaysIn(subs)
	if err != nil || len(essays) < 2 {
		t.Fatalf("ListRecentEssaysIn(%v) = %v,%v want essays (len > 2), nil", subs, essays, err)
	}

	// Test list essays in favor
	essay3 := mockEssay(user.ID)
	essay3.InReplyTo = sql.NullInt32{Int32: int32(essay2.ID), Valid: true}
	essay3.ReplyType = models.ReplyTypeSupports
	db.CreateEssay(essay3)
	// list
	essays, err = db.ListEssayReplies(essay2.ID, essay3.ReplyType)
	if err != nil || len(essays) != 1 {
		t.Fatalf("ListEssayReplies(%v,%v) = %v,%v want essays, nil", essay2.ID, essay3.ReplyType, essays, err)
	}

	// Clean
	toDelete := []*models.Essay{
		essay3,
		essay2,
		essay,
	}
	for _, es := range toDelete {
		err = db.DeleteEssay(es.ID)
		if err != nil {
			t.Fatalf("DeleteEssay(%v) = %v, want nil", es.ID, err)
		}
		db.DeleteSubdiscepto(es.PostedIn)
	}
	db.DeleteUser(user.ID)
}
func TestVotes(t *testing.T) {
	// Setup needed data
	user := mockUser()
	db.CreateUser(user, mockPasswd)
	essay := mockEssay(user.ID)
	db.CreateSubdiscepto(mockSubdiscepto(), user.ID)
	_ = db.CreateEssay(essay)

	// Actual test
	upvotes, downvotes, err := db.CountVotes(essay.ID)
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
	err = db.CreateVote(vote)
	if err != nil {
		t.Fatalf("CreateVote(%v) = %v, want nil", vote, err)
	}

	// Check added upvote
	upvotes, downvotes, err = db.CountVotes(essay.ID)
	if err != nil || upvotes == 0 || downvotes != 0 {
		t.Fatalf("CountVotes(%v) = %v,%v,%v, want upvotes, 0, nil",
			essay.ID, upvotes, downvotes, err)
	}

	// Delete (needed to change vote type for same user)
	err = db.DeleteVote(vote.EssayID, vote.UserID)
	if err != nil {
		t.Fatalf("DeleteVote(%v, %v) = %v, want nil",
			vote.EssayID, vote.UserID, err)
	}

	// Clean
	err = db.DeleteVote(vote.EssayID, vote.UserID)
	if err != nil {
		t.Fatalf("DeleteVote(%v, %v) = %v, want nil",
			vote.EssayID, vote.UserID, err)
	}

	db.DeleteEssay(essay.ID)
	db.DeleteUser(user.ID)
}
func TestSubdiscepto(t *testing.T) {
	// Setup needed data
	user := mockUser()
	db.CreateUser(user, mockPasswd)

	// Actual test
	subdis := &models.Subdiscepto{
		Name:        "subtest",
		Description: "here we talk about tests",
	}
	err := db.CreateSubdiscepto(subdis, user.ID)
	if err != nil {
		t.Fatalf("CreateSubdiscepto(%v, %v) = %v, want nil", subdis, user.ID, err)
	}

	subs, err := db.ListSubdisceptos()
	if err != nil || len(subs) == 0 {
		t.Fatalf("ListSubdisceptos() = %v, %v, want subs (len >= 1), nil", subdis, err)
	}

	// Join a sub
	user2 := mockUser()
	user2.Email += "as"

	db.CreateUser(user2, mockPasswd)

	err = db.JoinSubdiscepto(mockSubName, user2.ID)
	if err != nil {
		t.Fatalf("JoinSubdiscepto(%v,%v) = %v, want nil", mockSubName, user2.ID, err)
	}

	mySubs, err := db.ListMySubdisceptos(user2.ID)
	if mySubs[0] != mockSubName || err != nil {
		t.Fatalf("ListMySubdisceptos(%v) = %v,%v want mySubs, nil", user2.ID, mySubs, err)
	}

	err = db.LeaveSubdiscepto(mockSubName, user2.ID)
	if err != nil {
		t.Fatalf("LeaveSubdiscepto(%v,%v) = %v, want nil", mockSubName, user2.ID, err)
	}

	err = db.DeleteSubdiscepto(subdis.Name)
	if err != nil {
		t.Fatalf("DeleteSubdiscepto(%v) = %v, want nil", subdis.Name, err)
	}

	// Clean
	db.DeleteUser(user.ID)
	db.DeleteUser(user2.ID)
}
func TestSearch(t *testing.T) {
	user := mockUser()
	db.CreateUser(user, mockPasswd)
	db.CreateSubdiscepto(mockSubdiscepto(), user.ID)
	essay := mockEssay(user.ID)
	db.CreateEssay(essay)

	testValues := []struct {
		input []string
		want  int
	}{
		{[]string{"happy"}, 0},
		{[]string{"fruit"}, 1},
		{[]string{"banana"}, 1},
		{[]string{"banana", "best"}, 1},
		{[]string{"best"}, 1},
	}

	for _, v := range testValues {
		essays, err := db.SearchByTags(v.input)
		if err != nil || len(essays) != v.want {
			t.Fatalf(
				"SearchByTags(%v) = %v,%v, want len(essays) = %v, nil",
				v.input,
				essays,
				err,
				v.want,
			)
		}
	}

	// Clean
	db.DeleteEssay(essay.ID)
	db.DeleteUser(user.ID)
	db.DeleteSubdiscepto(mockSubName)
}
