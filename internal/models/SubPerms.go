package models

type SubPerms struct {
	EssayPerms
	CreateEssay       bool
	BanUser           bool
	DeleteSubdiscepto bool
	AddMod            bool
}

var SubPermsOwner SubPerms = SubPerms{
	EssayPerms: EssayPerms{
		true, true,
	},
	CreateEssay:       true,
	BanUser:           true,
	DeleteSubdiscepto: true,
	AddMod:            true,
}
