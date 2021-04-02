package models

type GlobalPerms struct {
	Login             bool
	CreateSubdiscepto bool
	BanUserGlobally   bool
	DeleteUser        bool
	ManageGlobalRole  bool
	SubPerms
}

func (self GlobalPerms) And(other GlobalPerms) GlobalPerms {
	return GlobalPerms{
		Login:             self.Login && other.Login,
		CreateSubdiscepto: self.CreateSubdiscepto && other.CreateSubdiscepto,
		BanUserGlobally:   self.BanUserGlobally && other.BanUserGlobally,
		DeleteUser:        self.DeleteUser && other.DeleteUser,
		ManageGlobalRole:  self.ManageGlobalRole && other.ManageGlobalRole,
		SubPerms:          self.SubPerms.And(other.SubPerms),
	}
}
