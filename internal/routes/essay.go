package routes

import (
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
	r.Get("/{id}", routes.AppHandler(routes.GetEssay))

	loggedIn := r.With(routes.EnforceCtx(UserHCtxKey))
	loggedIn.Put("/", routes.UpdateEssay)
	loggedIn.Delete("/{id}", routes.DeleteEssay)
	loggedIn.Post("/{essayID}/vote", routes.AppHandler(routes.PostVote))
}
func (routes *Routes) GetNewEssay(w http.ResponseWriter, r *http.Request) AppError {
	subdiscepto := r.URL.Query().Get("subdiscepto")

	user := r.Context().Value(UserHCtxKey).(*db.UserH)
	subs, err := user.ListMySubdisceptos(r.Context())

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	essay := struct {
		*models.Essay
		MySubdisceptos []string
	}{
		Essay: &models.Essay{
			PostedIn:  subdiscepto,
			InReplyTo: inReplyTo,
		},
		MySubdisceptos: subs,
	}

	routes.tmpls.RenderHTML(w, "newEssay", essay)
	return nil
}
func (routes *Routes) GetEssay(w http.ResponseWriter, r *http.Request) AppError {
	userH, _ := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH, _ := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	subData, err := subH.Read(r.Context())
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}
	esH, err := subH.GetEssayH(r.Context(), id, userH)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	refutesList, err := subH.ListReplies(r.Context(), *esH, &models.ReplyTypeRefutes.String)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	supportList, err := subH.ListReplies(r.Context(), *esH, &models.ReplyTypeSupports.String)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	generalList, err := subH.ListReplies(r.Context(), *esH, &models.ReplyTypeGeneral.String)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essay, err := esH.GetEssay(r.Context())
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	isMember := false
	var subs []string
	var essayUserDid *models.EssayUserDid
	if userH != nil {
		essayUserDid, err = esH.GetUserDid(r.Context(), *userH)
		if err != nil {
			return &ErrInternal{Cause: err}
		}
		subs, err = userH.ListMySubdisceptos(r.Context())
		if err != nil {
			return &ErrInternal{Cause: err}
		}

		for _, s := range subs {
			if s == subH.Name() {
				isMember = true
				break
			}
		}
	}

	data := struct {
		NameSubdiscepto string
		DescSubdiscepto string
		IsMember        bool
		Essay           *models.Essay
		RefutesList     []*models.Essay
		GeneralList     []*models.Essay
		SupportList     []*models.Essay
		EssayUserDid    *models.EssayUserDid
		SubdisceptoList []string
	}{
		NameSubdiscepto: subData.Name,
		DescSubdiscepto: subData.Description,
		IsMember:        isMember,
		Essay:           essay,
		EssayUserDid:    essayUserDid,
		SubdisceptoList: subs,
		RefutesList:     refutesList,
		GeneralList:     generalList,
		SupportList:     supportList,
	}

	routes.tmpls.RenderHTML(w, "essay", data)
	return nil
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) AppError {
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	disceptoH := r.Context().Value(DiscpetoHCtxKey).(*db.DisceptoH)

	subH, err := disceptoH.GetSubdisceptoH(r.Context(), r.FormValue("postedIn"), userH)
	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))

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

	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		Tags:           tags,
		AttributedToID: userH.ID(),
		PostedIn:       subH.Name(),
		InReplyTo:      inReplyTo,
		ReplyType:      replyType,
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
	userH := r.Context().Value(UserHCtxKey).(*db.UserH)
	subH := r.Context().Value(SubdisceptoHCtxKey).(*db.SubdisceptoH)

	essayIDStr := chi.URLParam(r, "essayID")
	essayID, err := strconv.Atoi(essayIDStr)

	var vote models.VoteType
	switch r.FormValue("vote") {
	case "upvote":
		vote = models.VoteTypeUpvote
	case "downvote":
		vote = models.VoteTypeDownvote
	}

	esH, err := subH.GetEssayH(r.Context(), essayID, userH)

	esH.DeleteVote(r.Context(), *userH)
	err = esH.CreateVote(r.Context(), *userH, vote)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	http.Redirect(w, r, fmt.Sprintf("/s/%s/%d", subdiscepto, essayID), http.StatusSeeOther)
	return nil
}
