package render

import (
	"bytes"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"gitlab.com/ranfdev/discepto/internal/models"
)

type Templates struct {
	templates *template.Template
	funcMap   template.FuncMap
	envConfig *models.EnvConfig
}

func (tmpls *Templates) RenderHTML(w http.ResponseWriter, tmplName string, data interface{}) {
	// Reload templates every time when developing locally.
	if tmpls.envConfig.Debug {
		tmpls.loadFromDisk()
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
func formatTime(args ...interface{}) template.HTML {
	t := args[0].(time.Time)
	return template.HTML(t.Format("Jan 2 15:04:05"))
}
func now(args ...interface{}) time.Time {
	return time.Now()
}
func (tmpls *Templates) loadFromDisk() {
	tmpls.templates = template.Must(
		template.New("").
			Funcs(tmpls.funcMap).
			ParseGlob("web/templates/*"),
	)
}
func (tmpls *Templates) SetFS(fs fs.FS) {
	tmpls.templates = template.Must(template.New("").
		Funcs(tmpls.funcMap).
		ParseFS(fs, "templates/*html"),
	)
}
func GetTemplates(envConfig *models.EnvConfig) Templates {
	tmpls := Templates{envConfig: envConfig}
	tmpls.funcMap = template.FuncMap{
		"markdown":        markdown,
		"markdownPreview": markdownPreview,
		"now":             now,
		"formatTime":      formatTime,
	}
	return tmpls
}
