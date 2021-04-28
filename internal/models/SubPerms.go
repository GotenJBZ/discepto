package models

type SubPerms struct {
	ReadSubdiscepto   bool
	UpdateSubdiscepto bool
	CreateEssay       bool
	DeleteEssay       bool
	BanUser           bool
	DeleteSubdiscepto bool
	ChangeRanking     bool
	ManageRole        bool
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
}

func (self SubPerms) And(other SubPerms) SubPerms {
	return SubPerms{
		ReadSubdiscepto:   self.ReadSubdiscepto && other.ReadSubdiscepto,
		UpdateSubdiscepto: self.UpdateSubdiscepto && other.UpdateSubdiscepto,
		CreateEssay:       self.CreateEssay && other.CreateEssay,
		DeleteEssay:       self.DeleteEssay && other.DeleteEssay,
		BanUser:           self.BanUser && other.BanUser,
		ChangeRanking:     self.ChangeRanking && other.ChangeRanking,
		DeleteSubdiscepto: self.DeleteSubdiscepto && other.DeleteSubdiscepto,
		ManageRole:        self.ManageRole && other.ManageRole,
	}
}
func SubPermsFromMap(m map[string]bool) SubPerms {
	return SubPerms{
		ReadSubdiscepto:   m["read_subdiscepto"],
		UpdateSubdiscepto: m["update_subdiscepto"],
		CreateEssay:       m["create_essay"],
		DeleteEssay:       m["delete_essay"],
		BanUser:           m["ban_user"],
		DeleteSubdiscepto: m["delete_subdiscepto"],
		ChangeRanking:     m["change_ranking"],
		ManageRole:        m["manage_role"],
	}
}
func SubPermsToMap(perms SubPerms) map[string]bool {
	// TODO: finish
	return map[string]bool{
		"read_subdiscepto":   perms.ReadSubdiscepto,
		"update_subdiscepto": perms.UpdateSubdiscepto,
		"create_essay":       perms.CreateEssay,
		"delete_essay":       perms.DeleteEssay,
		"ban_user":           perms.BanUser,
		"delete_subdiscepto": perms.DeleteSubdiscepto,
		"change_ranking":     perms.ChangeRanking,
		"manage_role":        perms.ManageRole,
	}
}
