package controller

//
//import (
//	"embed"
//	"html/template"
//	"io"
//	"net/http"
//
//	"github.com/labstack/echo/v4"
//)
//
////go:embed static/*
//var templates embed.FS
//
//var test = map[string]bool{
//	"key1": false,
//	"key2": true,
//}
//
//type Template struct {
//	templates *template.Template
//}
//
//func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
//	return t.templates.ExecuteTemplate(w, name, data)
//}
//
//func RegisterTemplates(e *echo.Echo) {
//	t := &Template{
//		templates: template.Must(template.ParseFS(templates, "static/*.html")),
//	}
//	e.Renderer = t
//	e.GET("/", func(c echo.Context) error {
//		return c.Render(http.StatusOK, "index", test)
//	})
//	e.GET("/approve/:key", ApproveKey)
//}
//
//func ApproveKey(c echo.Context) error {
//	key := c.Param("key")
//	test[key] = true
//	return c.Redirect(http.StatusTemporaryRedirect, "/")
//}
