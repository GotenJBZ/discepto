package server

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"os"
)

var DEBUG bool
var templates *template.Template

func init() {
	templates = initTemplates()
	if os.Getenv("DEBUG") == "true" {
		DEBUG = true
	} else {
		DEBUG = false
	}
}
func RenderHTML(w http.ResponseWriter, tmplName string, data interface{}) {
	t := getTemplates()
	buff := bytes.NewBuffer([]byte{})
	err := t.ExecuteTemplate(buff, tmplName, data)
	if err != nil && tmplName != "404" {
		RenderHTML(w, "404", nil)
		log.Println(err)
		return
	}
	w.Header().Add("Content-Type", "text/html")
	w.Write(buff.Bytes())
}
func initTemplates() *template.Template {
	return template.Must(template.ParseGlob("web/templates/*"))
}
func getTemplates() *template.Template {
	// Reload templates every time when developing locally.
	if DEBUG {
		return initTemplates()
	} else {
		// Use templates already in memory when in production (faster)
		return templates
	}
}
