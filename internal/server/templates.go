package server

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/markbates/pkger"
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
	template := template.New("")
	err := pkger.Walk("/web/templates/", func (path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			// skip dir
			return nil
		}
		file, err := pkger.Open(path)
		if err != nil {
			return err
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		_, err = template.Parse(string(content))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return template
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
