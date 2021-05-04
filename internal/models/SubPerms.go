package models

import "gitlab.com/ranfdev/discepto/internal/utils"

type SubPerms struct {
	ReadSubdiscepto   bool
	UpdateSubdiscepto bool
	CreateEssay       bool
	DeleteEssay       bool
	BanUser           bool
	DeleteSubdiscepto bool
	ChangeRanking     bool
	ManageRole        bool
	CommonAfterRejoin bool
}

var SubPermsOwner SubPerms = SubPerms{
	ReadSubdiscepto:   true,
	UpdateSubdiscepto: true,
	CreateEssay:       true,
	DeleteEssay:       true,
	BanUser:           true,
	ChangeRanking:     true,
	DeleteSubdiscepto: true,
	ManageRole:        true,
	CommonAfterRejoin: true,
}

func (self SubPerms) And(other SubPerms) SubPerms {
	return utils.StructAnd(self, other).(SubPerms)
}
func SubPermsFromMap(m map[string]bool) SubPerms {
	p := SubPerms{}
	utils.BoolMapToStruct(m, &p)
	return p
}
func (perms SubPerms) ToBoolMap() map[string]bool {
	return utils.StructToBoolMap(perms)
}
