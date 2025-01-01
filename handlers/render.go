package handlers

import (
	"io"
	"text/template"

	"github.com/labstack/echo/v4"
)

const views string = "views/*.html"

type Template struct {
	Templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.Templates.ExecuteTemplate(w, name, data)
}

func NewRenderer() *Template {
    return &Template{
        Templates: template.Must(template.ParseGlob(views)),
    }
}

