package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEmail(t *testing.T) {
	valid := []string{
		"rasdfs@gmail.com",
		"rasdfs@piosdf.com",
		"asdfj.jh@pio.sdf.com",
	}
	invalid := []string{
		"asdjfkjsdhf",
		"@asdfjaskh",
		"asdfasdf@",
	}

	for _, v := range valid {
		if !ValidateEmail(v) {
			t.Errorf("Email should be valid: %s", v)
		}
	}

	for _, v := range invalid {
		if ValidateEmail(v) {
			t.Errorf("Email should be invalid: %s", v)
		}
	}
}
func TestBoolMapStruct(t *testing.T) {
	require := require.New(t)
	type Three struct {
		FourtyTwo bool
	}
	s := struct {
		One bool
		Two bool
		Three
	}{
		One: true,
		Two: false,
		Three: Three{
			FourtyTwo: true,
		},
	}
	require.Equal(map[string]bool{
		"one":        true,
		"two":        false,
		"fourty_two": true,
	}, StructToBoolMap(s))
}
