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
	r.Post("/", routes.AppHandler(routes.PostEssay))
	r.Post("/{essayID}/vote", routes.AppHandler(routes.PostVote))
	r.Put("/", routes.UpdateEssay)
	r.Delete("/{id}", routes.DeleteEssay)
}
func (routes *Routes) GetNewEssay(w http.ResponseWriter, r *http.Request) AppError {
	subdiscepto := r.URL.Query().Get("subdiscepto")

	user := r.Context().Value("user").(*models.User)
	subs, err := routes.db.ListMySubdisceptos(user.ID)

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	essay := struct{
		EssayModel		models.Essay
		MySubdisceptos		[]string
	}{
		EssayModel: models.Essay{
			PostedIn:	subdiscepto,
			InReplyTo:	inReplyTo,
		},
		MySubdisceptos:	subs,
	}



	routes.tmpls.RenderHTML(w, "newEssay", essay)
	return nil
}
func (routes *Routes) PostEssay(w http.ResponseWriter, r *http.Request) AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &ErrMustLogin{}
	}

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
		AttributedToID: user.ID,
		PostedIn:       r.FormValue("postedIn"),
		InReplyTo:      inReplyTo,
		ReplyType:      replyType,
	}
	err = routes.db.CreateEssay(&essay)
	if err == db.ErrBadContentLen {
		return &ErrBadRequest{Cause: err, Motivation: "You must respect required content length"}
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
	user, ok := r.Context().Value("user").(*models.User)
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

	routes.db.DeleteVote(essayID, user.ID)
	err = routes.db.CreateVote(&models.Vote{
		UserID:   user.ID,
		EssayID:  essayID,
		VoteType: vote,
	})
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	essay, err := routes.db.GetEssay(essayID)
	if err != nil {
		return &ErrInternal{Cause: err}
	}

	http.Redirect(w, r, fmt.Sprintf("/s/%s/%d", essay.PostedIn, essayID), http.StatusSeeOther)
	return nil
}
