package routes

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"gitlab.com/ranfdev/discepto/internal/db"
	"gitlab.com/ranfdev/discepto/internal/models"
)

func (routes *Routes) EssaysRouter(r chi.Router) {
	specificEssay := r.With(routes.EssayCtx)
	specificEssay.Get("/{essayID}", routes.AppHandler(routes.GetEssay))
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Put("/{essayID}", routes.UpdateEssay)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Delete("/{essayID}", routes.DeleteEssay)
	specificEssay.With(routes.EnforceCtx(UserHCtxKey)).Post("/{essayID}/vote", routes.AppHandler(routes.PostVote))
}
func (routes *Routes) EssayCtx(next http.Handler) http.Handler {
	return routes.AppHandler(func(w http.ResponseWriter, r *http.Request) AppError {
		userH := GetUserH(r)
		subH := GetSubdisceptoH(r)

		essayIDStr := chi.URLParam(r, "essayID")
		essayID, err := strconv.Atoi(essayIDStr)
		if err != nil {
			return &ErrBadRequest{Cause: err}
		}

		esH, err := subH.GetEssayH(r.Context(), essayID, userH)
		if err != nil {
			return &ErrInternal{Cause: err}
		}
		ctx := context.WithValue(r.Context(), EssayHCtxKey, esH)
		next.ServeHTTP(w, r.WithContext(ctx))
		return nil
	})
}
func (routes *Routes) GetNewEssay(w http.ResponseWriter, r *http.Request) AppError {
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
	return nil
}
func (routes *Routes) GetEssay(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	subH := GetSubdisceptoH(r)
	esH, _ := r.Context().Value(EssayHCtxKey).(*db.EssayH)

	subData, err := subH.ReadView(r.Context(), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	filter := r.URL.Query().Get("replyType")
	if filter == "" {
		filter = "general"
	}
	replies, err := subH.ListReplies(r.Context(), *esH, &filter)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essay, err := esH.ReadView(r.Context())
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	subs, err := userH.ListMySubdisceptos(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essayUserDid := &models.EssayUserDid{}
	if userH != nil {
		essayUserDid, err = esH.GetUserDid(r.Context(), *userH)
		if err != nil {
			return &ErrInternal{Cause: err}
		}
	}

	data := struct {
		Subdiscepto     *models.SubdisceptoView
		Essay           *models.EssayView
		Replies         []models.EssayView
		FilterReplyType string
		Sources         []string
		EssayUserDid    *models.EssayUserDid
		SubdisceptoList []string
	}{
		Subdiscepto:     subData,
		Essay:           essay,
		EssayUserDid:    essayUserDid,
		SubdisceptoList: subs,
		Sources:         []string{},
		Replies:         replies,
		FilterReplyType: filter,
	}

	routes.tmpls.RenderHTML(w, "essay", data)
	return nil
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	disceptoH := GetDisceptoH(r)

	subH, err := disceptoH.GetSubdisceptoH(r.Context(), r.FormValue("postedIn"), userH)
	if err != nil {
		return &ErrInternal{Cause: err}
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
			return &ErrInternal{Cause: err}
		}
		_, err = subH.CreateEssayReply(r.Context(), &essay, *parentH)
	} else {
		_, err = subH.CreateEssay(r.Context(), &essay)
	}

	if err == db.ErrBadContentLen {
		return &ErrBadRequest{
			Cause:      err,
			Motivation: "You must respect required content length",
		}
	} else if err != nil {
		return &ErrInternal{Cause: err}
	}

	http.Redirect(w, r, fmt.Sprintf("/s/%s", essay.PostedIn), http.StatusSeeOther)
	return nil
}
func (routes *Routes) DeleteEssay(w http.ResponseWriter, r *http.Request) {

}
func (routes *Routes) UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
func (routes *Routes) PostVote(w http.ResponseWriter, r *http.Request) AppError {
	userH := GetUserH(r)
	esH, _ := r.Context().Value(EssayHCtxKey).(*db.EssayH)

	var vote models.VoteType
	switch r.FormValue("vote") {
	case "upvote":
		vote = models.VoteTypeUpvote
	case "downvote":
		vote = models.VoteTypeDownvote
	}

	esH.DeleteVote(r.Context(), *userH)
	err := esH.CreateVote(r.Context(), *userH, vote)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	essayID := chi.URLParam(r, "essayID")
	http.Redirect(w, r, fmt.Sprintf("/s/%s/%s", subdiscepto, essayID), http.StatusSeeOther)
	return nil
}
