package models

import (
	"errors"

	"gitlab.com/ranfdev/discepto/internal/utils"
)

var ErrPermDenied = errors.New("Missing permissions to execute action")

type GlobalPerms struct {
	CreateSubdiscepto   bool
	BanUserGlobally     bool
	UseLocalPermissions bool
	DeleteUser          bool
	ManageGlobalRole    bool
	SubPerms
}

func (self GlobalPerms) And(other GlobalPerms) GlobalPerms {
	return utils.StructAnd(self, other).(GlobalPerms)
}

func GlobalPermsFromMap(m map[string]bool) GlobalPerms {
	p := GlobalPerms{}
	utils.BoolMapToStruct(m, &p)
	return p
}
func (perms GlobalPerms) ToBoolMap() map[string]bool {
	return utils.StructToBoolMap(perms)
}
