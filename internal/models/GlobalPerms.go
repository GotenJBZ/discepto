package models

type GlobalPerms struct {
	Login             bool
	CreateSubdiscepto bool
	BanUserGlobally   bool
	DeleteUser        bool
	AddAdmin          bool
}
