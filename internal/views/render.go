package views

import (
	"bytes"
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static
var statics embed.FS

//go:embed templates
var templates embed.FS

var parsedTemplates = template.Must(template.ParseFS(
	templates,
	"templates/*.html",
	"templates/partials/*.html",
))

func Render(name string, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := parsedTemplates.ExecuteTemplate(&buf, name, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Static() fs.FS {
	return statics
}
