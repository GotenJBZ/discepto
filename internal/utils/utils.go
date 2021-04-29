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
func BoolMapToStruct(bm map[string]bool, into interface{}) {
	v := reflect.ValueOf(into).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name := ToSnakeCase(t.Field(i).Name)
		switch v.Field(i).Type().Kind() {
		case reflect.Bool:
			v.Field(i).Set(reflect.ValueOf(bm[name]))
		case reflect.Struct:
			BoolMapToStruct(bm, v.Field(i).Addr().Interface())
		}
	}
}
func StructAnd(s1 interface{}, s2 interface{}) interface{} {
	vs1 := reflect.ValueOf(s1)
	vs2 := reflect.ValueOf(s2)
	out := reflect.New(reflect.ValueOf(s1).Type()).Elem()

	if vs1.Type() != vs2.Type() {
		panic("can't run AND on different types")
	}
	for i := 0; i < vs1.NumField(); i++ {
		switch vs1.Field(i).Type().Kind() {
		case reflect.Bool:
			v := vs1.Field(i).Bool() && vs2.Field(i).Bool()
			out.Field(i).Set(reflect.ValueOf(v))
		case reflect.Struct:
			out.Field(i).Set(reflect.ValueOf(StructAnd(vs1.Field(i), vs2.Field(i))))
		}
	}
	return out.Interface()
}
func StructToBoolMap(s interface{}, pMap ...*map[string]bool) map[string]bool {
	vs1 := reflect.ValueOf(s)
	ts1 := vs1.Type()
	m := &map[string]bool{}
	if len(pMap) > 0 {
		m = pMap[0]
	}
	for i := 0; i < vs1.NumField(); i++ {
		switch vs1.Field(i).Type().Kind() {
		case reflect.Bool:
			(*m)[ToSnakeCase(ts1.Field(i).Name)] = vs1.Field(i).Bool()
		case reflect.Struct:
			StructToBoolMap(vs1.Field(i).Interface(), m)
		}
	}
	return *m
}
