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

const mockPasswd = "correcthorsebatterystaple!3" // hackerman
const mockSubName = "mock"
const mockSubName2 = "mock2"

func mockUser() *models.User {
	return &models.User{
		Name:  "User1",
		Email: "pippo@strana.com",
	}
}
func mockUser2() *models.User {
	return &models.User{
		Name:  "User2",
		Email: "asdfasdf@fasdf.com",
	}
}
func mockUrl() url.URL {
	url, _ := url.Parse("https://example.com")
	return *url
}
func mockEssay(userID int) *models.Essay {
	replyData := models.Replying{
		ReplyType: models.ReplyTypeGeneral,
	}
	return &models.Essay{
		Thesis: "Banana is the best fruit",
		Content: `Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...
		Banana is the best fruit because...`,
		AttributedToID: userID, // it's a reference, can't mock this
		Published:      time.Now(),
		PostedIn:       mockSubName,
		Replying:       replyData,
		Tags:           []string{"banana", "fruit", "best"},
		Sources:        []url.URL{mockUrl()},
	}
}
func mockSubdisceptoReq() *models.SubdisceptoReq {
	return &models.SubdisceptoReq{
		Name:        mockSubName,
		Description: "Mock subdiscepto",
		Public:      true,
	}
}
func mockSubdisceptoReq2() *models.SubdisceptoReq {
	return &models.SubdisceptoReq{
		Name:        mockSubName2,
		Description: "Mock subdiscepto 2",
		Public:      true,
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
	ctx := context.Background()

	err := execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
		db := db.withTx(tx)
		passwd := mockPasswd
		userH, err := db.CreateUser(context.Background(), mockUser(), passwd)
		require.Nil(err)
		disceptoH, err := db.GetDisceptoH(context.Background(), userH)
		require.Nil(err)
		users, err := disceptoH.ListMembers(context.Background())
		require.Nil(err)
		require.Len(users, 1)

		user2 := mockUser2()
		user2.Email = "asdfhkjhhuiu"
		testData := []struct {
			user *models.User
			err  error
		}{
			{mockUser(), models.ErrEmailAlreadyUsed},
			{user2, models.ErrInvalidFormat},
		}

		for _, td := range testData {
			_, err := db.CreateUser(context.Background(), td.user, passwd)
			require.Equal(td.err, err)
		}

		err = userH.Delete(context.Background())
		require.Nil(err)
		return nil
	})
	require.Nil(err)
}
func TestAuth(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	ctx := context.Background()
	user := mockUser()
	passwd := mockPasswd

	err := execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
		db := db.withTx(tx)
		_, err := db.CreateUser(ctx, user, passwd)
		require.Nil(err)

		var userH *UserH
		{
			// With a bad passwd
			passwd = "93sdjfhkasdhfkjha"
			_, err := db.Login(ctx, user.Email, passwd)
			require.NotNil(err)
		}
		{
			// Normal login
			passwd = mockPasswd
			uH, err := db.Login(ctx, user.Email, passwd)
			userH = uH
			require.Nil(err)
		}

		// Clean
		require.Nil(userH.Delete(ctx))
		return nil
	})
	require.Nil(err)
}
func TestEssay(t *testing.T) {
	require := require.New(t)
	t.Parallel()
	ctx := context.Background()

	err := execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
		db := db.withTx(tx)
		user := mockUser()
		userH, err := db.CreateUser(ctx, user, mockPasswd)
		require.Nil(err)

		disceptoH, err := db.GetDisceptoH(ctx, userH)
		require.Nil(err)
		subH, err := disceptoH.CreateSubdiscepto(ctx, *userH, mockSubdisceptoReq())
		require.Nil(err)

		essay := mockEssay(user.ID)
		essayH, err := subH.CreateEssay(ctx, essay)
		require.Nil(err)

		essays, err := subH.ListEssays(ctx)
		require.NotNil(essays)
		require.Nil(err)

		// Test list recent essays from joined subs
		// Create and fill second sub
		sub2H, err := disceptoH.CreateSubdiscepto(ctx, *userH, mockSubdisceptoReq2())
		require.Nil(err)
		essay2 := mockEssay(user.ID)
		essay2.PostedIn = mockSubName2
		essay2H, err := sub2H.CreateEssay(ctx, essay2)
		require.Nil(err)

		// list
		subs := []string{mockSubName, mockSubName2}
		recentEssays, err := disceptoH.ListRecentEssaysIn(ctx, subs)
		require.Nil(err)
		require.Len(recentEssays, 2)

		// Test list essays in favor
		essay3 := mockEssay(user.ID)
		essay3.InReplyTo = sql.NullInt32{Int32: int32(essay2.ID), Valid: true}
		essay3.ReplyType = models.ReplyTypeSupports
		parentEssayH, err := sub2H.GetEssayH(ctx, essay2.ID, userH)
		require.Nil(err)
		_, err = sub2H.CreateEssayReply(ctx, essay3, *parentEssayH)
		require.Nil(err)

		// Create upvote
		err = essayH.CreateVote(ctx, *userH, models.VoteTypeUpvote)
		require.Nil(err)
		updatedEssay, err := essayH.ReadView(ctx)
		require.Nil(err)
		require.Equal(1, updatedEssay.Upvotes)
		require.Equal(0, updatedEssay.Downvotes)

		// Delete vote
		err = essayH.DeleteVote(ctx, *userH)
		require.Nil(err)

		// Create downvote
		err = essayH.CreateVote(ctx, *userH, models.VoteTypeDownvote)
		require.Nil(err)
		updatedEssay, err = essayH.ReadView(ctx)
		require.Nil(err)
		require.Equal(0, updatedEssay.Upvotes)
		require.Equal(1, updatedEssay.Downvotes)

		// Check what a specific user did
		did, err := essayH.GetUserDid(ctx, *userH)
		require.Nil(err)
		require.Equal(&models.EssayUserDid{
			Vote: sql.NullString{String: string(models.VoteTypeDownvote), Valid: true},
		}, did)

		// list
		essayReplies, err := sub2H.ListReplies(ctx, *essay2H, &models.ReplyTypeSupports.String)
		require.Nil(err)
		require.Len(essayReplies, 1)

		// Clean
		err = subH.Delete(ctx)
		require.Nil(err)
		err = sub2H.Delete(ctx)
		require.Nil(err)
		require.Nil(userH.Delete(ctx))
		return nil
	})
	require.Nil(err)
}

func TestSubdiscepto(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	ctx := context.Background()

	err := execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
		db := db.withTx(tx)
		{
			// user1
			// Setup needed data
			user := mockUser()
			userH, err := db.CreateUser(ctx, user, mockPasswd)
			require.Nil(err)

			subdis := mockSubdisceptoReq()
			disceptoH, err := db.GetDisceptoH(ctx, userH)
			require.Nil(err)

			_, err = disceptoH.CreateSubdiscepto(ctx, *userH, subdis)
			require.Nil(err)

			subs, err := db.ListSubdisceptos(ctx, userH)
			require.Nil(err)
			require.Len(subs, 1)
		}
		{
			// user2
			// Join a sub
			user := mockUser2()

			userH, err := db.CreateUser(ctx, user, mockPasswd)
			require.Nil(err)

			disceptoH, err := db.GetDisceptoH(ctx, userH)
			require.Nil(err)
			subH, err := disceptoH.GetSubdisceptoH(ctx, mockSubdisceptoReq().Name, userH)
			require.Nil(err)
			err = subH.AddMember(ctx, *userH)
			require.Nil(err)

			mySubs, err := userH.ListMySubdisceptos(ctx)
			require.Nil(err)
			require.Len(mySubs, 1)
			require.Equal(mockSubName, mySubs[0])

			err = subH.RemoveMember(ctx, *userH)
			require.Nil(err)

			// Delete (should fail, because user2 doesn't have that permission)
			err = subH.Delete(ctx)
			require.NotNil(err)

			require.Nil(userH.Delete(ctx))
		}

		// Clean
		userH, err := db.Login(ctx, mockUser().Email, mockPasswd)
		require.Nil(err)
		disceptoH, err := db.GetDisceptoH(ctx, userH)
		require.Nil(err)
		subH, err := disceptoH.GetSubdisceptoH(ctx, mockSubName, userH)
		require.Nil(err)
		err = subH.Delete(ctx)
		require.Nil(err)

		require.Nil(userH.Delete(ctx))
		return nil
	})
	require.Nil(err)
}
func TestSearch(t *testing.T) {
	require := require.New(t)

	user := mockUser()
	userH, err := db.CreateUser(context.Background(), user, mockPasswd)
	require.Nil(err)
	disceptoH, err := db.GetDisceptoH(context.Background(), userH)
	require.Nil(err)
	subH, err := disceptoH.CreateSubdiscepto(context.Background(), *userH, mockSubdisceptoReq())
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
	t.Parallel()
	require := require.New(t)
	user := mockUser()
	ctx := context.Background()

	err := execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
		db := db.withTx(tx)
		// Create necessary entities
		userH, err := db.CreateUser(ctx, user, mockPasswd)
		defer userH.Delete(ctx)

		require.Nil(err)
		disceptoH, err := db.GetDisceptoH(ctx, userH)
		require.Nil(err)
		subH, err := disceptoH.CreateSubdiscepto(ctx, *userH, mockSubdisceptoReq())
		require.Nil(err)
		defer subH.Delete(ctx)

		user2 := mockUser2()
		user2H, err := db.CreateUser(ctx, user2, mockPasswd)
		require.Nil(err)
		defer user2H.Delete(ctx)

		err = subH.AddMember(ctx, *user2H)
		require.Nil(err)

		table := []struct {
			Name string
			Func func(db SharedDB) func(t *testing.T)
		}{
			{"Check default permissions", testRolesDefaultPerms},
			{"Ban user from subdiscepto", testRolesBanUserFromSub},
			{"Ban user globally", testRolesBanUserGlobally},
		}

		for _, r := range table {
			execTx(ctx, db.db, func(ctx context.Context, tx DBTX) error {
				db := db.withTx(tx)
				t.Run(r.Name, r.Func(db))
				return nil
			})
		}
		return nil
	})
	require.Nil(err)
}
func testRolesDefaultPerms(db SharedDB) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		require := require.New(t)

		// Test first user (admin)
		{
			userH, _ := db.Login(ctx, mockUser().Email, mockPasswd)
			disceptoH, _ := db.GetDisceptoH(ctx, userH)
			subH, _ := disceptoH.GetSubdisceptoH(ctx, mockSubName, userH)

			// Check "admin" global role
			require.Equal(models.PermsGlobalAdmin, disceptoH.Perms())

			// Check "admin" sub role
			require.Equal(models.PermsSubAdmin.Union(disceptoH.Perms()), subH.Perms())
		}
		// Test second user (common)
		{
			user2H, _ := db.Login(ctx, mockUser2().Email, mockPasswd)
			discepto2H, _ := db.GetDisceptoH(ctx, user2H)
			sub2H, _ := discepto2H.GetSubdisceptoH(ctx, mockSubName, user2H)

			// Check "common" global role
			require.Equal(models.PermsGlobalCommon, discepto2H.Perms())

			// Check "common" sub role
			require.Equal(models.PermsSubCommon.Union(discepto2H.Perms()), sub2H.Perms())
		}
	}
}
func testRolesBanUserFromSub(db SharedDB) func(t *testing.T) {
	return func(t *testing.T) {
		require := require.New(t)
		ctx := context.Background()

		userH, err := db.Login(ctx, mockUser().Email, mockPasswd)
		disceptoH, err := db.GetDisceptoH(ctx, userH)
		user2H, err := db.Login(ctx, mockUser2().Email, mockPasswd)
		subH, err := disceptoH.GetSubdisceptoH(ctx, mockSubName, userH)

		// Remove "common" global role, banning the user
		roleH, err := subH.GetRoleH(ctx, "common")
		require.Nil(err)
		err = subH.Unassign(ctx, user2H.id, *roleH)
		require.Nil(err)
		subPerms2, err := getUserPerms(ctx, db.db, subH.RoleDomain(), user2H.id)
		require.Nil(err)
		require.Equal(models.NewPerms(), subPerms2)

		// A banned user shouldn't be able to leave the subdiscepto without a trace.
		// The membership track record must be kept, to ensure the user stays banned
		dis2H, err := db.GetDisceptoH(ctx, user2H)
		sub2H, err := dis2H.GetSubdisceptoH(ctx, subH.Name(), user2H)
		require.Nil(err)
		err = sub2H.RemoveMember(ctx, *user2H)
		require.Nil(err)
		members, err := sub2H.ListMembers(ctx)
		require.Nil(err)
		found := false
		for _, m := range members {
			if m.UserID == user2H.id {
				found = true
				break
			}
		}
		require.True(found)
	}
}
func testRolesBanUserGlobally(db SharedDB) func(t *testing.T) {
	return func(t *testing.T) {
		require := require.New(t)
		ctx := context.Background()
		user2ID := 0
		{
			user2H, _ := db.Login(ctx, mockUser2().Email, mockPasswd)
			user2ID = user2H.id
		}

		// As User1, unassign all roles to User2
		{
			userH, _ := db.Login(ctx, mockUser().Email, mockPasswd)
			disceptoH, _ := db.GetDisceptoH(ctx, userH)

			err := disceptoH.UnassignAll(ctx, user2ID)
			require.Nil(err)
		}
		// As User2, list own permissions
		{
			user2H, _ := db.Login(ctx, mockUser2().Email, mockPasswd)
			disceptoH, err := db.GetDisceptoH(ctx, user2H)
			require.Nil(err)

			// User is banned, so it shouldn't have any permission
			require.Equal(models.NewPerms(), disceptoH.Perms())

			// The user doesn't have "use_local_permissions" so it shouldn't be able
			// to do anything inside a subdiscepto
			// TODO: the use should be able to get a subdisceptoh with the public permissions
			_, err = disceptoH.GetSubdisceptoH(ctx, mockSubName, user2H)
			require.IsType(models.ErrMissingPerms{}, err)
		}

	}
}
