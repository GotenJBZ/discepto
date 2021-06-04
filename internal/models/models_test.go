package models

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPerms(t *testing.T) {
	require := require.New(t)
	ps := NewPerms(PermUpdateSubdiscepto)
	err := ps.Require(
		PermUpdateSubdiscepto,
		PermCreateSubdiscepto,
	)
	require.Error(err)

	ps2 := NewPerms(PermUpdateSubdiscepto, PermCreateSubdiscepto)
	require.True(ps.SubsetOf(ps2))
	require.False(ps2.SubsetOf(ps))
}
