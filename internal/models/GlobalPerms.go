package models

type GlobalPerms struct {
	ID                int
	CanLogin          bool
	CreateSubdiscepto bool
	BanUserGlobally   bool
	AddAdmin          bool
}
