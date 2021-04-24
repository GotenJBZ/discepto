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
