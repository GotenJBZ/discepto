package utils

import "testing"

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
func TestCheckPerms(t *testing.T) {
	type Entry struct {
		provide string
		need    string
		expect  bool
	}
	entries := []Entry{
		{
			provide: "",
			need:    "",
			expect:  true,
		},
		{
			provide: "adsfj asdfakj",
			need:    "",
			expect:  true,
		},
		{
			provide: "delete_posts",
			need:    "delete_posts",
			expect:  true,
		},
		{
			provide: "delete_posts ban_users",
			need:    "ban_users delete_posts",
			expect:  true,
		},
		{
			provide: "",
			need:    "delete_posts ban_users",
			expect:  false,
		},
		{
			provide: "adsfj asdfakj",
			need:    "delete_posts ban_users",
			expect:  false,
		},
		{
			provide: "delete_posts,ban_users",
			need:    "delete_posts ban_users",
			expect:  false, // must use space, not comma
		},
		{
			provide: "delete_posts",
			need:    "ban_users delete_posts",
			expect:  false,
		},
	}
	for _, e := range entries {
		if CheckPerms(e.provide, e.need) != e.expect {
			t.Errorf("Expecting %t: provided '%s', needed '%s'", e.expect, e.provide, e.need)
		}
	}

}
