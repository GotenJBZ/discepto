package models

import (
	"errors"
	"fmt"
)

var ErrPermDenied = errors.New("Missing permissions to execute action")

type Perm string
type Perms map[Perm]struct{}

func NewPerms(perms ...Perm) Perms {
	ps := Perms{}
	for _, p := range perms {
		ps[p] = struct{}{}
	}
	return ps
}

const (
	PermCreateSubdiscepto   Perm = "create_subdiscepto"
	PermReadSubdiscepto     Perm = "read_subdiscepto"
	PermUpdateSubdiscepto   Perm = "update_subdiscepto"
	PermDeleteSubdiscepto   Perm = "delete_subdiscepto"
	PermDeleteUser          Perm = "delete_user"
	PermReadEssay           Perm = "read_essay"
	PermCreateEssay         Perm = "create_essay"
	PermDeleteEssay         Perm = "delete_essay"
	PermChangeRanking       Perm = "change_ranking"
	PermCommonAfterRejoin   Perm = "common_after_rejoin"
	PermCreateReport        Perm = "create_report"
	PermViewReport          Perm = "view_report"
	PermDeleteReport        Perm = "delete_report"
	PermUseLocalPermissions Perm = "use_local_permissions"
	PermManageGlobalRole    Perm = "manage_global_role"
	PermManageRole          Perm = "manage_role"
	PermBanUserGlobally     Perm = "ban_user_globally"
	PermBanUser             Perm = "ban_user"
	PermCreateVote          Perm = "create_vote"
	PermDeleteVote          Perm = "delete_vote"
)

var PermsSubAdmin = NewPerms(
	PermReadSubdiscepto,
	PermUpdateSubdiscepto,
	PermCreateEssay,
	PermDeleteEssay,
	PermBanUser,
	PermChangeRanking,
	PermDeleteSubdiscepto,
	PermManageRole,
	PermCommonAfterRejoin,
	PermCreateReport,
	PermViewReport,
	PermDeleteReport,
)

var PermsGlobalAdmin = NewPerms(
	PermReadSubdiscepto,
	PermCreateSubdiscepto,
	PermUpdateSubdiscepto,
	PermCreateEssay,
	PermDeleteEssay,
	PermBanUser,
	PermBanUserGlobally,
	PermChangeRanking,
	PermDeleteSubdiscepto,
	PermManageRole,
	PermCommonAfterRejoin,
	PermCreateReport,
	PermViewReport,
	PermDeleteReport,
	PermManageGlobalRole,
	PermManageGlobalRole,
	PermManageGlobalRole,
	PermDeleteUser,
	PermUseLocalPermissions,
	PermCreateVote,
	PermDeleteVote,
)

var PermsGlobalCommon = NewPerms(
	PermUseLocalPermissions,
	PermCreateVote,
	PermDeleteVote,
)

var PermsSubCommon = NewPerms(
	PermReadSubdiscepto,
	PermCreateEssay,
	PermCommonAfterRejoin,
	PermCreateReport,
)

type ErrMissingPerms struct {
	Perms []Perm
}

func (mp ErrMissingPerms) Error() string {
	return fmt.Sprintf("missing permission %s", mp.Perms)
}

func (ps Perms) Require(reqPerms ...Perm) error {
	missing := []Perm{}
	for _, p := range reqPerms {
		if _, ok := ps[p]; !ok {
			missing = append(missing, p)
			return ErrMissingPerms{missing}
		}
	}
	return nil
}

func (ps Perms) RequirePerms(reqPerms Perms) error {
	missing := []Perm{}
	for p := range reqPerms {
		if _, ok := ps[p]; !ok {
			missing = append(missing, p)
			return ErrMissingPerms{missing}
		}
	}
	return nil
}

func (ps Perms) Check(reqPerms ...Perm) bool {
	return ps.Require(reqPerms...) == nil
}

func (ps Perms) List() []Perm {
	perms := []Perm{}
	for k := range ps {
		perms = append(perms, k)
	}
	return perms
}

func (ps Perms) SubsetOf(ps2 Perms) bool {
	for p := range ps {
		if _, ok := ps2[p]; !ok {
			return false
		}
	}
	return true
}
func (ps Perms) Union(ps2 Perms) Perms {
	allPerms := []Perm{}
	for p := range ps {
		allPerms = append(allPerms, p)
	}
	for p := range ps2 {
		allPerms = append(allPerms, p)
	}
	return NewPerms(allPerms...)
}

func (ps Perms) Intersect(ps2 Perms) Perms {
	res := Perms{}
	for p := range ps {
		if _, ok := ps2[p]; ok {
			res[p] = struct{}{}
		}
	}
	return res
}
