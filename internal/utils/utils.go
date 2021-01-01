package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
)

func ValidateEmail(email string) bool {
	regex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	return regex.MatchString(email)
}
func CheckPerms(provided string, needed string) (ok bool) {
	if needed == "" {
		return true
	}
	prov := strings.Fields(provided)
	need := strings.Fields(needed)
	for _, n := range need {
		ok = false
		for _, p := range prov {
			if n == p {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return ok
}
func GenToken(l int) string {
	randBytes := make([]byte, l)
	_, err := rand.Read(randBytes)
	log.Fatal().AnErr("Generating random token", err)
	return hex.EncodeToString(randBytes)
}
