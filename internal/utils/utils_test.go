package utils

import (
	"testing"
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
