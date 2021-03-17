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
	r.Post("/{essayID}/vote", routes.AppHandler(routes.PostVote))
	r.Put("/", routes.UpdateEssay)
	r.Delete("/{id}", routes.DeleteEssay)
}
func (routes *Routes) GetNewEssay(w http.ResponseWriter, r *http.Request) AppError {
	subdiscepto := r.URL.Query().Get("subdiscepto")

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	essay := models.Essay{
		PostedIn:  subdiscepto,
		InReplyTo: inReplyTo,
	}
	routes.tmpls.RenderHTML(w, "newEssay", essay)
	return nil
}
func (routes *Routes) GetEssay(w http.ResponseWriter, r *http.Request) AppError {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	user, _ := r.Context().Value("user").(*db.UserH)
	subH, err := routes.db.GetSubdisceptoH(subdiscepto, user)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essayH, err := subH.GetEssayH(id)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay, err := essayH.GetEssay()
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay.Upvotes, essay.Downvotes, err = essayH.CountVotes()
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	routes.tmpls.RenderHTML(w, "essay", essay)
	return nil
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*db.UserH)
	if !ok {
		return &ErrMustLogin{}
	}

	postedIn := r.FormValue("postedIn")
	subH, err := routes.db.GetSubdisceptoH(postedIn, user)

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	// Parse reply type
	replyType := r.FormValue("replyType")
	found := false
	for _, t := range models.AvailableReplyTypes {
		if replyType == t {
			found = true
			break
		}
	}
	if !found {
		replyType = models.ReplyTypeGeneral
	}

	// Parse tags
	tags := strings.Fields(r.FormValue("tags"))

	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		Tags:           tags,
		AttributedToID: user.ID(),
		PostedIn:       postedIn,
		InReplyTo:      inReplyTo,
		ReplyType:      replyType,
	}
	_, err = subH.CreateEssay(&essay)
	if err == db.ErrBadContentLen {
		return &ErrBadRequest{
			Cause:      err,
			Motivation: "You must respect required content length",
		}
	}
	if err != nil {
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
	user, ok := r.Context().Value("user").(*db.UserH)
	if !ok {
		return &ErrMustLogin{}
	}

	essayIDStr := chi.URLParam(r, "essayID")
	essayID, err := strconv.Atoi(essayIDStr)

	var vote models.VoteType
	switch r.FormValue("vote") {
	case "upvote":
		vote = models.VoteTypeUpvote
	case "downvote":
		vote = models.VoteTypeDownvote
	}

	subH, err := routes.db.GetSubdisceptoH(chi.URLParam(r, "subdiscepto"), user)
	essayH, err := subH.GetEssayH(essayID)

	essayH.DeleteVote()
	err = essayH.CreateVote(&models.Vote{
		UserID:   user.ID(),
		EssayID:  essayID,
		VoteType: vote,
	})
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	http.Redirect(w, r, fmt.Sprintf("/s/%s/%d", subdiscepto, essayID), http.StatusSeeOther)
	return nil
}
