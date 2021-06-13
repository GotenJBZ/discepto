package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequire(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	table := []struct {
		Given    []Perm
		Required []Perm
		IsErr    bool
	}{
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{PermUpdateSubdiscepto},
			false,
		},
		{
			[]Perm{PermUpdateSubdiscepto, PermCommonAfterRejoin},
			[]Perm{PermUpdateSubdiscepto},
			false,
		},
		{
			[]Perm{PermUpdateSubdiscepto, PermDeleteUser, PermReadEssay},
			[]Perm{PermUpdateSubdiscepto},
			false,
		},
		{
			[]Perm{},
			[]Perm{},
			false,
		},
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{},
			false,
		},
		{
			[]Perm{},
			[]Perm{PermUpdateSubdiscepto},
			true,
		},
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{PermUpdateSubdiscepto, PermCreateEssay},
			true,
		},
		{
			[]Perm{PermCreateEssay},
			[]Perm{PermUpdateSubdiscepto, PermCreateEssay},
			true,
		},
	}

	for _, r := range table {
		ps := NewPerms(r.Given...)
		err := ps.Require(r.Required...)
		require.Equal(r.IsErr, err != nil, fmt.Sprintf("given=%v, required=%v", r.Given, r.Required))
	}

}

func TestSubset(t *testing.T) {
	t.Parallel()
	require := require.New(t)
	table := []struct {
		P1       []Perm
		P2       []Perm
		IsSubset bool
	}{
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{PermUpdateSubdiscepto},
			true,
		},
		{
			[]Perm{PermUpdateSubdiscepto, PermCommonAfterRejoin},
			[]Perm{PermUpdateSubdiscepto},
			true,
		},
		{
			[]Perm{PermUpdateSubdiscepto, PermDeleteUser, PermReadEssay},
			[]Perm{PermUpdateSubdiscepto},
			true,
		},
		{
			[]Perm{},
			[]Perm{},
			true,
		},
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{},
			true,
		},
		{
			[]Perm{},
			[]Perm{PermUpdateSubdiscepto},
			false,
		},
		{
			[]Perm{PermUpdateSubdiscepto},
			[]Perm{PermUpdateSubdiscepto, PermCreateEssay},
			false,
		},
		{
			[]Perm{PermCreateEssay},
			[]Perm{PermUpdateSubdiscepto, PermCreateEssay},
			false,
		},
	}
	for _, r := range table {
		p1 := NewPerms(r.P1...)
		p2 := NewPerms(r.P2...)
		ok := p2.SubsetOf(p1)
		require.Equal(r.IsSubset, ok, fmt.Sprintf("p1=%v, p2=%v", p1, p2))
	}
}
