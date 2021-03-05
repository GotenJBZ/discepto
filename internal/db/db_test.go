package db

import (
	"database/sql"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
	require := require.New(t)
	user2 := mockUser()
	user2.Email = "asdasdasdfjh"
	testData := []struct {
		user *models.User
		err  error
	}{
		{mockUser(), nil},
		{mockUser(), ErrEmailAlreadyUsed},
		{user2, ErrInvalidFormat},
	}

	passwd := mockPasswd
	for _, td := range testData {
		err := db.CreateUser(td.user, passwd)
		require.Equal(err, td.err)

	}

	users, err := db.ListUsers()
	require.Nil(err)
	require.Len(users, 1)

	for _, td := range testData {
		if td.err == nil {
			require.Nil(db.DeleteUser(td.user.ID))
		}
	}
}
func TestAuth(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	passwd := mockPasswd
	err := db.CreateUser(user, passwd)
	require.Nil(err)

	// With a bad passwd
	passwd = "93sdjfhkasdhfkjha"
	token, err := db.Login(user.Email, passwd)
	require.NotNil(err)

	// Normal login
	passwd = mockPasswd
	token, err = db.Login(user.Email, passwd)
	require.Nil(err)

	// Retrieve user by token
	user2, err := db.GetUserByToken(token)
	require.Nil(err)
	require.Equal(user.ID, user2.ID)

	// Sign out
	require.Nil(db.Signout(token))

	// Clean
	require.Nil(db.DeleteUser(user.ID))
}
func TestEssay(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	require.Nil(db.CreateUser(user, mockPasswd))
	require.Nil(db.CreateSubdiscepto(mockSubdiscepto(), user.ID))

	essay := mockEssay(user.ID)
	err := db.CreateEssay(essay)
	require.Nil(err)

	essays, err := db.ListEssays(mockSubName)
	require.NotNil(essays)
	require.Nil(err)

	// Test list recent essays from joined subs
	// Create and fill second sub
	require.Nil(db.CreateSubdiscepto(mockSubdiscepto2(), user.ID))
	essay2 := mockEssay(user.ID)
	essay2.PostedIn = mockSubName2
	require.Nil(db.CreateEssay(essay2))

	// list
	subs := []string{mockSubName, mockSubName2}
	essays, err = db.ListRecentEssaysIn(subs)
	require.Nil(err)
	require.Len(essays, 2)

	// Test list essays in favor
	essay3 := mockEssay(user.ID)
	essay3.InReplyTo = sql.NullInt32{Int32: int32(essay2.ID), Valid: true}
	essay3.ReplyType = models.ReplyTypeSupports
	require.Nil(db.CreateEssay(essay3))
	// list
	essays, err = db.ListEssayReplies(essay2.ID, essay3.ReplyType)
	require.Nil(err)
	require.Len(essays, 1)

	// Clean
	toDelete := []*models.Essay{
		essay3,
		essay2,
		essay,
	}
	for _, es := range toDelete {
		require.Nil(db.DeleteEssay(es.ID))
		require.Nil(db.DeleteSubdiscepto(es.PostedIn))
	}
	require.Nil(db.DeleteUser(user.ID))
}

func TestVotes(t *testing.T) {
	require := require.New(t)

	// Setup needed data
	user := mockUser()
	require.Nil(db.CreateUser(user, mockPasswd))

	essay := mockEssay(user.ID)
	require.Nil(db.CreateSubdiscepto(mockSubdiscepto(), user.ID))
	require.Nil(db.CreateEssay(essay))

	// Actual test
	upvotes, downvotes, err := db.CountVotes(essay.ID)
	require.Nil(err)
	require.Equal(upvotes, 0)
	require.Equal(downvotes, 0)

	// Add upvote
	vote := &models.Vote{
		UserID:   user.ID,
		EssayID:  essay.ID,
		VoteType: models.VoteTypeUpvote,
	}
	require.Nil(db.CreateVote(vote))

	// Check added upvote
	upvotes, downvotes, err = db.CountVotes(essay.ID)
	require.Nil(err)
	require.Equal(upvotes, 1)
	require.Equal(downvotes, 0)

	// Delete (needed to change vote type for same user)
	require.Nil(db.DeleteVote(vote.EssayID, vote.UserID))

	// Add downvote
	vote = &models.Vote{
		UserID:   user.ID,
		EssayID:  essay.ID,
		VoteType: models.VoteTypeDownvote,
	}
	require.Nil(db.CreateVote(vote))
	upvotes, downvotes, err = db.CountVotes(essay.ID)
	require.Nil(err)
	require.Equal(upvotes, 0)
	require.Equal(downvotes, 1)

	// Clean
	require.Nil(db.DeleteVote(vote.EssayID, vote.UserID))
	require.Nil(db.DeleteEssay(essay.ID))
	require.Nil(db.DeleteSubdiscepto(mockSubName))
	require.Nil(db.DeleteUser(user.ID))
}
func TestSubdiscepto(t *testing.T) {
	require := require.New(t)
	// Setup needed data
	user := mockUser()
	require.Nil(db.CreateUser(user, mockPasswd))

	// Actual test
	subdis := mockSubdiscepto()

	err := db.CreateSubdiscepto(subdis, user.ID)
	require.Nil(err)

	subs, err := db.ListSubdisceptos()
	require.Nil(err)
	require.Len(subs, 1)

	// Join a sub
	user2 := mockUser()
	user2.Email += "as"

	require.Nil(db.CreateUser(user2, mockPasswd))

	err = db.JoinSubdiscepto(mockSubName, user2.ID)
	require.Nil(err)

	mySubs, err := db.ListMySubdisceptos(user2.ID)
	require.Nil(err)
	require.Len(mySubs, 1)
	require.Equal(mySubs[0], mockSubName)

	err = db.LeaveSubdiscepto(mockSubName, user2.ID)
	require.Nil(err)

	err = db.DeleteSubdiscepto(subdis.Name)
	require.Nil(err)

	// Clean
	require.Nil(db.DeleteUser(user.ID))
	require.Nil(db.DeleteUser(user2.ID))
}
func TestSearch(t *testing.T) {
	require := require.New(t)

	user := mockUser()
	require.Nil(db.CreateUser(user, mockPasswd))
	require.Nil(db.CreateSubdiscepto(mockSubdiscepto(), user.ID))
	essay := mockEssay(user.ID)
	require.Nil(db.CreateEssay(essay))

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
		require.Nil(err)
		require.Len(essays, v.want)
	}

	// Clean
	require.Nil(db.DeleteEssay(essay.ID))
	require.Nil(db.DeleteSubdiscepto(mockSubName))
	require.Nil(db.DeleteUser(user.ID))
}
func TestSubPerms(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	require.Nil(db.CreateUser(user, "asdfasdf"))
	require.Nil(db.CreateSubdiscepto(mockSubdiscepto(), user.ID))

	perms, err := db.GetSubPerms(user.ID, mockSubName)
	require.Nil(err)
	require.Equal(perms, models.SubPermsOwner)

	require.Nil(db.DeleteSubdiscepto(mockSubName))
	require.Nil(db.DeleteUser(user.ID))
}
