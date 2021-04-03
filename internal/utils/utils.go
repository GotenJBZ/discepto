package utils

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
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

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func ToSnakeCase(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
func ParseFormStruct(r *http.Request, into interface{}) error {
	v := reflect.ValueOf(into).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name := ToSnakeCase(f.Name)
		switch k := f.Type.Kind(); k {
		case reflect.Int:
			intg, err := strconv.Atoi(r.FormValue(name))
			if err != nil {
				return err
			}
			v.Field(i).Set(reflect.ValueOf(intg))
		case reflect.Bool:
			isChecked := r.FormValue(name) == "on"
			v.Field(i).Set(reflect.ValueOf(isChecked))
		case reflect.String:
			v.Field(i).Set(reflect.ValueOf(r.FormValue(name)))
		}
	}
	return nil
}
