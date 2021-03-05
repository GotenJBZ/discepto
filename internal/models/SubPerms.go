package models

type SubPerms struct {
	DeleteEssay       bool
	CreateEssay       bool
	BanUser           bool
	ChangeRanking     bool
	DeleteSubdiscepto bool
	AddMod            bool
}

var SubPermsOwner SubPerms = SubPerms{
	true,
	true,
	true,
	true,
	true,
	true,
}
