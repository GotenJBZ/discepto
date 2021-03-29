package models

type SubPerms struct {
	Read              bool
	CreateEssay       bool
	DeleteEssay       bool
	BanUser           bool
	DeleteSubdiscepto bool
	AssignRoles       bool
}

var SubPermsOwner SubPerms = SubPerms{
	Read:              true,
	CreateEssay:       true,
	DeleteEssay:       true,
	BanUser:           true,
	DeleteSubdiscepto: true,
	AssignRoles:       true,
}
