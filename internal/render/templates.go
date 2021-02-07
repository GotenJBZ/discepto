package render

import (
	"bytes"
	"html/template"
	"log"
	"net/http"

	"gitlab.com/ranfdev/discepto/internal/models"
)

type Templates struct {
	templates *template.Template
	envConfig *models.EnvConfig
}

func (tmpls *Templates) RenderHTML(w http.ResponseWriter, tmplName string, data interface{}) {
	// Reload templates every time when developing locally.
	if tmpls.envConfig.Debug {
		tmpls.load()
	}
	buff := bytes.NewBuffer([]byte{})
	err := tmpls.templates.ExecuteTemplate(buff, tmplName, data)
	if err != nil && tmplName != "404" {
		tmpls.RenderHTML(w, "404", nil)
		log.Println(err)
		return
	}
	w.Header().Add("Content-Type", "text/html")
	w.Write(buff.Bytes())
}
func (tmpls *Templates) load() {
	tmpls.templates = template.Must(template.ParseGlob("web/templates/*"))
}
func GetTemplates(envConfig *models.EnvConfig) Templates {
	tmpls := Templates{envConfig: envConfig}
	tmpls.load()
	return tmpls
}
