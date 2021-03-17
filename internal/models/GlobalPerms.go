package models

type GlobalPerms struct {
	ID                int
	Login             bool
	CreateSubdiscepto bool
	BanUserGlobally   bool
	DeleteUser        bool
	AddAdmin          bool
}
