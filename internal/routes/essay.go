package routes

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) EssaysRouter(r chi.Router) {
	specificEssay := r.With(routes.EssayCtx)
	specificEssay.Get("/{essayID}", routes.GetEssay)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Put("/{essayID}", routes.UpdateEssay)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Delete("/{essayID}", routes.DeleteEssay)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Post("/{essayID}/vote", routes.PostVote)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Post("/{essayID}/report", routes.PostReport)
}
func (routes *Routes) EssayCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userH := GetUserH(r)
		subH := GetSubdisceptoH(r)

		essayIDStr := chi.URLParam(r, "essayID")
		essayID, err := strconv.Atoi(essayIDStr)
		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}

		esH, err := subH.GetEssayH(r.Context(), essayID, userH)
		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), EssayHCtxKey, esH)
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}
func (routes *Routes) GetNewEssay(w http.ResponseWriter, r *http.Request) {
	subdiscepto := r.URL.Query().Get("subdiscepto")

	userH := GetUserH(r)
	subs, err := userH.ListMySubdisceptos(r.Context())

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	essay := struct {
		*models.Essay
		MySubdisceptos []string
	}{
		Essay: &models.Essay{
			PostedIn: subdiscepto,
			Replying: models.Replying{
				InReplyTo: inReplyTo,
			},
		},
		MySubdisceptos: subs,
	}

	routes.tmpls.RenderHTML(w, "newEssay", essay)
	return
}
func (routes *Routes) GetEssay(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	subH := GetSubdisceptoH(r)
	disceptoH := GetDisceptoH(r)
	esH, _ := r.Context().Value(EssayHCtxKey).(*db.EssayH)

	repliesCount, err := esH.CountReplies(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	subData, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	filter := r.URL.Query().Get("replyType")
	if filter == "" {
		filter = "general"
	}
	replies, err := subH.ListReplies(r.Context(), *esH, &filter)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	essay, err := esH.ReadView(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	subs, err := userH.ListMySubdisceptos(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	essayUserDid := &models.EssayUserDid{}
	if userH != nil {
		essayUserDid, err = esH.GetUserDid(r.Context(), *userH)
		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}
	}

	user, err := disceptoH.ReadPublicUser(r.Context(), essay.AttributedToID)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	data := struct {
		Subdiscepto     *models.SubdisceptoView
		Essay           *models.EssayView
		Replies         []models.EssayView
		RepliesCount    map[string]int
		FilterReplyType string
		Sources         []string
		EssayUserDid    *models.EssayUserDid
		SubdisceptoList []string
		Perms           models.Perms
		User            *models.UserView
	}{
		Subdiscepto:     subData,
		Essay:           essay,
		EssayUserDid:    essayUserDid,
		SubdisceptoList: subs,
		Sources:         []string{},
		Replies:         replies,
		RepliesCount:    repliesCount,
		FilterReplyType: filter,
		Perms:           esH.Perms().Union(subH.Perms()),
		User:            user,
	}

	routes.tmpls.RenderHTML(w, "essay", data)
	return
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)

	subH, err := disceptoH.GetSubdisceptoH(r.Context(), r.FormValue("postedIn"), userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	rep, err := strconv.Atoi(r.FormValue("inReplyTo"))

	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	// Parse reply type
	replyType := models.ReplyTypeGeneral
	{
		rType := r.FormValue("replyType")
		for _, t := range models.AvailableReplyTypes {
			if rType == t.String {
				replyType = sql.NullString{
					String: rType,
					Valid:  true,
				}
				break
			}
		}
	}

	// Parse tags
	tags := strings.Fields(r.FormValue("tags"))

	replyData := models.Replying{
		InReplyTo: inReplyTo,
		ReplyType: replyType,
	}
	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		AttributedToID: userH.ID(),
		PostedIn:       subH.Name(),
		Replying:       replyData,
		Tags:           tags,
	}

	// Finally create the essay
	// If it's a reply, check if the user can actually see the parent essay
	if inReplyTo.Valid {
		parentH, err := subH.GetEssayH(r.Context(), int(inReplyTo.Int32), userH)
		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}
		_, err = subH.CreateEssayReply(r.Context(), &essay, *parentH)
	} else {
		_, err = subH.CreateEssay(r.Context(), &essay)
	}

	if err == models.ErrBadContentLen {
		err := &ErrBadRequest{
			Cause:      err,
			Motivation: "You must respect required content length",
		}
		routes.HandleErr(w, r, err)
		return
	} else if err != nil {
		routes.HandleErr(w, r, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/s/%s", essay.PostedIn), http.StatusSeeOther)
	return
}
func (routes *Routes) PostReport(w http.ResponseWriter, r *http.Request) {
	essayH := GetEssayH(r)
	userH := GetUserH(r)
	report := models.Report{}
	report.EssayID = essayH.ID()
	report.FromUserID = userH.ID()
	err := essayH.CreateReport(r.Context(), report, *userH)
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	return
}
func (routes *Routes) DeleteEssay(w http.ResponseWriter, r *http.Request) {
	essayH := GetEssayH(r)
	err := essayH.DeleteEssay(r.Context())
	if err != nil {
		routes.HandleErr(w, r, err)
		return
	}
	w.Header().Add("HX-Redirect", path.Dir(r.URL.Path))
	http.Redirect(w, r, path.Dir(r.URL.Path), http.StatusAccepted)
	return
}
func (routes *Routes) UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
func (routes *Routes) PostVote(w http.ResponseWriter, r *http.Request) {
	userH := GetUserH(r)
	esH := GetEssayH(r)
	userDid, err := esH.GetUserDid(r.Context(), *userH)
	if err != nil {
		routes.HandleErr(w,r,err)
	}

	var vote models.VoteType
	switch r.FormValue("vote") {
	case "upvote":
		vote = models.VoteTypeUpvote
	case "downvote":
		vote = models.VoteTypeDownvote
	}

	if userDid.Vote.Valid {
		err := esH.DeleteVote(r.Context(), *userH)
		if err != nil {
			routes.HandleErr(w, r, err)
		}
	}
	if models.VoteType(userDid.Vote.String) != vote {
		err = esH.CreateVote(r.Context(), *userH, vote)
		if err != nil {
			routes.HandleErr(w, r, err)
			return
		}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	essayID := chi.URLParam(r, "essayID")
	http.Redirect(w, r, fmt.Sprintf("/s/%s/%s", subdiscepto, essayID), http.StatusSeeOther)
	return
}
