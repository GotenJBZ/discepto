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

	user := r.Context().Value("user").(*db.UserH)
	subs, err := user.ListMySubdisceptos()

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
	user, ok := r.Context().Value("user").(*db.UserH)
	subdiscepto := chi.URLParam(r, "subdiscepto")
	subH, err := routes.db.GetSubdisceptoH(subdiscepto, user)

	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}
	esH, err := subH.GetEssayH(id, *user)
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay, err := esH.GetEssay()
	if err != nil {
		return &ErrNotFound{Cause: err, Thing: "essay"}
	}

	essay.Upvotes, essay.Downvotes, err = esH.CountVotes()
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	var subs []string
	if ok {
		subs, err = user.ListMySubdisceptos()
	}

	data := struct {
		Essay           *models.Essay
		SubdisceptoList []string
	}{
		Essay:           essay,
		SubdisceptoList: subs,
	}

	routes.tmpls.RenderHTML(w, "essay", data)
	return nil
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := db.ToUserH(r.Context().Value("user"))
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

	// Finally create the essay
	// If it's a reply, check if the user can actually see the parent essay
	if inReplyTo.Valid {
		parentH, err := subH.GetEssayH(int(inReplyTo.Int32), *user)
		if err != nil {
			return &ErrInternal{Cause: err}
		}
		_, err = subH.CreateEssayReply(&essay, *parentH)
	} else {
		_, err = subH.CreateEssay(&essay)
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
	esH, err := subH.GetEssayH(essayID, *user)

	esH.DeleteVote(*user)
	err = esH.CreateVote(*user, vote)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	subdiscepto := chi.URLParam(r, "subdiscepto")
	http.Redirect(w, r, fmt.Sprintf("/s/%s/%d", subdiscepto, essayID), http.StatusSeeOther)
	return nil
}
