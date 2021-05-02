package render

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/yuin/goldmark"
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
func markdown(args ...interface{}) template.HTML {
	var b bytes.Buffer
	s := args[0].(string)
	goldmark.Convert([]byte(s), &b)
	return template.HTML(b.String())
}
func markdownPreview(args ...interface{}) template.HTML {
	var b bytes.Buffer
	s := args[0].(string)
	i := strings.Index(s, "\n\r")
	maxLen := len(s)
	if 300 < maxLen {
		maxLen = 300
	}
	if i < 0 || i > maxLen {
		i = maxLen
	}
	goldmark.Convert([]byte(s[0:i]), &b)
	html := b.String()

	return template.HTML(string(html))
}
func (tmpls *Templates) load() {
	tmpls.templates = template.Must(template.New("").Funcs(template.FuncMap{
		"markdown":        markdown,
		"markdownPreview": markdownPreview,
	}).ParseGlob("web/templates/*"),
	)
}
func GetTemplates(envConfig *models.EnvConfig) Templates {
	tmpls := Templates{envConfig: envConfig}
	tmpls.load()
	return tmpls
}
