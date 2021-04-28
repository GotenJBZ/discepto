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

func GlobalPermsFromMap(m map[string]bool) GlobalPerms {
	return GlobalPerms{
		Login:             m["login"],
		CreateSubdiscepto: m["create_subdiscepto"],
		BanUserGlobally:   m["ban_user_globally"],
		ManageGlobalRole:  m["manage_global_role"],
		DeleteUser:        m["delete_user"],
		SubPerms:          SubPermsFromMap(m),
	}
}
func GlobalPermsToMap(perms GlobalPerms) map[string]bool {
	// TODO: finish
	m := map[string]bool{
		"login":              perms.Login,
		"create_subdiscepto": perms.CreateSubdiscepto,
		"ban_user_globally":  perms.BanUserGlobally,
		"manage_global_role": perms.ManageGlobalRole,
		"delete_user":        perms.DeleteUser,
	}
	sm := SubPermsToMap(perms.SubPerms)
	for k := range sm {
		m[k] = sm[k]
	}
	return m
}
