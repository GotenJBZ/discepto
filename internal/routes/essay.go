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
	"gitlab.com/ranfdev/discepto/internal/server"
)

func EssaysRouter(r chi.Router) {
	r.Post("/", AppHandler(PostEssay))
	r.Post("/{essayID}/vote", AppHandler(PostVote))
	r.Put("/", UpdateEssay)
	r.Delete("/{id}", DeleteEssay)
}
func GetNewEssay(w http.ResponseWriter, r *http.Request) *AppError {
	subdiscepto := r.URL.Query().Get("subdiscepto")

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	essay := models.Essay{
		PostedIn:  subdiscepto,
		InReplyTo: inReplyTo,
	}
	server.RenderHTML(w, "newEssay", essay)
	return nil
}
func PostEssay(w http.ResponseWriter, r *http.Request) *AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &AppError{Message: "Must login to execute this action"}
	}

	rep, err := strconv.Atoi(r.URL.Query().Get("inReplyTo"))
	inReplyTo := sql.NullInt32{Int32: int32(rep), Valid: err == nil}

	tags := strings.Fields(r.FormValue("tags"))
	essay := models.Essay{
		Thesis:         r.FormValue("thesis"),
		Content:        r.FormValue("content"),
		Tags:           tags,
		AttributedToID: user.ID,
		PostedIn:       r.FormValue("postedIn"),
		InReplyTo:      inReplyTo,
		ReplyType:      models.ParseReplyType(r.FormValue("replyType")),
	}
	err = db.CreateEssay(&essay)
	if err == db.ErrBadContentLen {
		return &AppError{Cause: err, Message: "You must respect required content length"}
	}
	if err != nil {
		return &AppError{Cause: err}
	}
	http.Redirect(w, r, fmt.Sprintf("/s/%s", essay.PostedIn), http.StatusSeeOther)
	return nil
}
func DeleteEssay(w http.ResponseWriter, r *http.Request) {

}
func UpdateEssay(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Nope")
}
func PostVote(w http.ResponseWriter, r *http.Request) *AppError {
	user, ok := r.Context().Value("user").(*models.User)
	if !ok {
		return &AppError{Message: "Must login"}
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

	db.DeleteVote(essayID, user.ID)
	err = db.CreateVote(&models.Vote{
		UserID:   user.ID,
		EssayID:  essayID,
		VoteType: vote,
	})
	if err != nil {
		return &AppError{Cause: err}
	}

	essay, err := db.GetEssay(essayID)
	if err != nil {
		return &AppError{Cause: err}
	}

	http.Redirect(w, r, fmt.Sprintf("/s/%s/%d", essay.PostedIn, essayID), http.StatusSeeOther)
	return nil
}
