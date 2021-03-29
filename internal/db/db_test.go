package db

import (
	"context"
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

var db SharedDB

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

	passwd := mockPasswd
	userH, err := db.CreateUser(context.Background(), mockUser(), passwd)
	require.Nil(err)
	disceptoH := db.GetDisceptoH(context.Background(), userH)
	users, err := disceptoH.ListUsers(context.Background())
	require.Nil(err)
	require.Len(users, 1)

	user2 := mockUser()
	user2.Email = "asdasdasdfjh"
	testData := []struct {
		user *models.User
		err  error
	}{
		{mockUser(), ErrEmailAlreadyUsed},
		{user2, ErrInvalidFormat},
	}

	for _, td := range testData {
		_, err := db.CreateUser(context.Background(), td.user, passwd)
		require.Equal(td.err, err)
	}

	err = userH.Delete(context.Background())
	require.Nil(err)
}
func TestAuth(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	passwd := mockPasswd
	_, err := db.CreateUser(context.Background(), user, passwd)
	require.Nil(err)

	// With a bad passwd
	passwd = "93sdjfhkasdhfkjha"
	token, err := db.Login(context.Background(), user.Email, passwd)
	require.NotNil(err)

	// Normal login
	passwd = mockPasswd
	token, err = db.Login(context.Background(), user.Email, passwd)
	require.Nil(err)

	// Retrieve user by token
	userH, err := db.GetUserH(context.Background(), token)
	require.Nil(err)
	require.Equal(user.ID, userH.id)

	// Sign out
	require.Nil(db.Signout(context.Background(), token))

	// Clean
	require.Nil(userH.Delete(context.Background()))
}
func TestEssay(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	userH, err := db.CreateUser(context.Background(), user, mockPasswd)
	require.Nil(err)

	disceptoH := db.GetDisceptoH(context.Background(), userH)
	subH, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdiscepto())
	require.Nil(err)

	essay := mockEssay(user.ID)
	_, err = subH.CreateEssay(context.Background(), essay)
	require.Nil(err)

	essays, err := subH.ListEssays(context.Background())
	require.NotNil(essays)
	require.Nil(err)

	// Test list recent essays from joined subs
	// Create and fill second sub
	sub2H, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdiscepto2())
	require.Nil(err)
	essay2 := mockEssay(user.ID)
	essay2.PostedIn = mockSubName2
	_, err = sub2H.CreateEssay(context.Background(), essay2)
	require.Nil(err)

	// list
	subs := []string{mockSubName, mockSubName2}
	essays, err = db.ListRecentEssaysIn(context.Background(), subs)
	require.Nil(err)
	require.Len(essays, 2)

	// Test list essays in favor
	essay3 := mockEssay(user.ID)
	essay3.InReplyTo = sql.NullInt32{Int32: int32(essay2.ID), Valid: true}
	essay3.ReplyType = models.ReplyTypeSupports
	parentEssayH, err := sub2H.GetEssayH(context.Background(), essay2.ID, *userH)
	require.Nil(err)
	_, err = sub2H.CreateEssayReply(context.Background(), essay3, *parentEssayH)
	require.Nil(err)
	//// list
	//essays, err = db.ListEssayReplies(essay2.ID, essay3.ReplyType)
	//require.Nil(err)
	//require.Len(essays, 1)

	// Clean
	err = subH.Delete(context.Background())
	require.Nil(err)
	err = sub2H.Delete(context.Background())
	require.Nil(err)
	require.Nil(userH.Delete(context.Background()))
}

func TestVotes(t *testing.T) {
	require := require.New(t)

	// Setup needed data
	user := mockUser()
	userH, err := db.CreateUser(context.Background(), user, mockPasswd)
	require.Nil(err)

	disceptoH := db.GetDisceptoH(context.Background(), userH)

	essay := mockEssay(user.ID)
	subH, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdiscepto())
	require.Nil(err)
	esH, err := subH.CreateEssay(context.Background(), essay)
	require.Nil(err)

	// Actual test
	upvotes, downvotes, err := esH.CountVotes(context.Background())
	require.Nil(err)
	require.Equal(0, upvotes)
	require.Equal(0, downvotes)

	// Add upvote
	require.Nil(esH.CreateVote(context.Background(), *userH, models.VoteTypeUpvote))

	// Check added upvote
	upvotes, downvotes, err = esH.CountVotes(context.Background())
	require.Nil(err)
	require.Equal(1, upvotes)
	require.Equal(0, downvotes)

	// Delete (needed to change vote type for same user)
	require.Nil(esH.DeleteVote(context.Background(), *userH))

	// Add downvote
	require.Nil(esH.CreateVote(context.Background(), *userH, models.VoteTypeDownvote))
	upvotes, downvotes, err = esH.CountVotes(context.Background())
	require.Nil(err)
	require.Equal(0, upvotes)
	require.Equal(1, downvotes)

	// Clean
	require.Nil(esH.DeleteVote(context.Background(), *userH))
	require.Nil(esH.DeleteEssay(context.Background()))
	require.Nil(subH.Delete(context.Background()))
	require.Nil(userH.Delete(context.Background()))
}
func TestSubdiscepto(t *testing.T) {
	require := require.New(t)
	{
		// user1
		// Setup needed data
		user := mockUser()
		userH, err := db.CreateUser(context.Background(), user, mockPasswd)
		require.Nil(err)

		subdis := mockSubdiscepto()
		disceptoH := db.GetDisceptoH(context.Background(), userH)

		_, err = disceptoH.CreateSubdiscepto(context.Background(), *userH, subdis)
		require.Nil(err)

		subs, err := db.ListSubdisceptos(context.Background())
		require.Nil(err)
		require.Len(subs, 1)
	}
	{
		// user2
		// Join a sub
		user := mockUser()
		user.Email += "as"

		userH, err := db.CreateUser(context.Background(), user, mockPasswd)
		require.Nil(err)

		subH, err := db.GetSubdisceptoH(context.Background(), mockSubdiscepto().Name, userH)
		require.Nil(err)
		err = userH.JoinSub(context.Background(), *subH)
		require.Nil(err)

		mySubs, err := userH.ListMySubdisceptos(context.Background())
		require.Nil(err)
		require.Len(mySubs, 1)
		require.Equal(mockSubName, mySubs[0])

		err = userH.LeaveSub(context.Background(), *subH)
		require.Nil(err)

		// Delete (should fail, because user2 doesn't have that permission)
		err = subH.Delete(context.Background())
		require.NotNil(err)

		require.Nil(userH.Delete(context.Background()))
	}

	// Clean
	token, err := db.Login(context.Background(), mockUser().Email, mockPasswd)
	require.Nil(err)
	userH, err := db.GetUserH(context.Background(), token)
	require.Nil(err)
	subH, err := db.GetSubdisceptoH(context.Background(), mockSubName, &userH)
	require.Nil(err)
	err = subH.Delete(context.Background())
	require.Nil(err)

	require.Nil(userH.Delete(context.Background()))
}
func TestSearch(t *testing.T) {
	require := require.New(t)

	user := mockUser()
	userH, err := db.CreateUser(context.Background(), user, mockPasswd)
	require.Nil(err)
	disceptoH := db.GetDisceptoH(context.Background(), userH)
	subH, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdiscepto())
	require.Nil(err)
	essay := mockEssay(user.ID)
	esH, err := subH.CreateEssay(context.Background(), essay)
	require.Nil(err)

	// testValues := []struct {
	// 	input []string
	// 	want  int
	// }{
	// 	{[]string{"happy"}, 0},
	// 	{[]string{"fruit"}, 1},
	// 	{[]string{"banana"}, 1},
	// 	{[]string{"banana", "best"}, 1},
	// 	{[]string{"best"}, 1},
	// }

	// for _, v := range testValues {
	// 	essays, err := db.SearchByTags(v.input)
	// 	require.Nil(err)
	// 	require.Len(essays, v.want)
	// }

	// Clean
	require.Nil(esH.DeleteEssay(context.Background()))
	require.Nil(subH.Delete(context.Background()))
	require.Nil(userH.Delete(context.Background()))
}
func TestRoles(t *testing.T) {
	require := require.New(t)
	user := mockUser()
	userH, err := db.CreateUser(context.Background(), user, mockPasswd)
	require.Nil(err)
	disceptoH := db.GetDisceptoH(context.Background(), userH)
	subH, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdiscepto())
	require.Nil(err)

	user2 := user
	user2.Email = "asdfasdf@fasdf.com"
	user2H, err := db.CreateUser(context.Background(), user2, mockPasswd)
	require.Nil(err)

	err = user2H.JoinSub(context.Background(), *subH)
	require.Nil(err)

	globalPerms := getGlobalPerms(context.Background(), db.db, userH)
	require.Equal(models.GlobalPerms{
		Login:             true,
		CreateSubdiscepto: true,
		DeleteUser:        true,
		BanUserGlobally:   true,
		AssignGlobalRoles: true,
	}, globalPerms)
	globalPerms2 := getGlobalPerms(context.Background(), db.db, user2H)
	require.Equal(models.GlobalPerms{
		Login:             true,
		CreateSubdiscepto: false,
		DeleteUser:        false,
		BanUserGlobally:   false,
		AssignGlobalRoles: false,
	}, globalPerms2)

	subPerms, err := getSubPerms(context.Background(), db.db, subH.subdiscepto, *userH)
	require.Equal(&models.SubPermsOwner, subPerms)
	require.Nil(err)

	subPerms2, err := getSubPerms(context.Background(), db.db, subH.subdiscepto, *user2H)
	require.Nil(err)
	require.Equal(&models.SubPerms{
		Read:              true,
		CreateEssay:       true,
		DeleteEssay:       false,
		DeleteSubdiscepto: false,
		BanUser:           false,
		AssignRoles:       false,
	}, subPerms2)
}
