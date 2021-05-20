package domain

type Report struct {
	ID          int
	Description string
	EssayID     int `db:"essay_id"`
	FromUserID  int `db:"from_user_id"`
	Essay
}
type ReportView struct {
	ID          int
	Description string
	EssayView   EssayView
}
