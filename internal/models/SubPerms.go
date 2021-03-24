package models

type SubPerms struct {
	Read bool
	EssayPerms
	CreateEssay       bool
	BanUser           bool
	DeleteSubdiscepto bool
	AddMod            bool
}

var SubPermsOwner SubPerms = SubPerms{
	Read: true,
	EssayPerms: EssayPerms{
		true, true, true,
	},
	CreateEssay:       true,
	BanUser:           true,
	DeleteSubdiscepto: true,
	AddMod:            true,
}
