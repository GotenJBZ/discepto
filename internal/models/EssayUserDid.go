package models

import "database/sql"

type EssayUserDid struct {
	Favourite bool
	Vote      sql.NullString
}
