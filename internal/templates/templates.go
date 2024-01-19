package templates

import (
	"embed"
	"text/template"
)

type TemplateHandler struct {
	tmplHome      *template.Template
	tmplSearch    *template.Template
	tmplLogin     *template.Template
	tmplSignUp    *template.Template
	tmplDashboard *template.Template
	tmplCreate    *template.Template
}

var templates embed.FS

func NewTemplateHandler(
	tmplHome *template.Template,
	tmplSearch *template.Template,
	tmplLogin *template.Template,
	tmplSignUp *template.Template,
	tmplDashboard *template.Template,
	tmplCreate *template.Template) *TemplateHandler {
	loggedoutTemplates := []string{"templates/index.html", "templates/header.html"}
	loggedinTemplates := []string{"templates/index.html", "templates/loggedinheader.html"}
	homeTemplate := "template/content.html"
	searchTemplate := "template/search.html"
	loginTemplate := "template/login.html"
	signupTemplate := "template/signup.html"
	dashboardTemplate := "template/dashboard.html"
	createTemplate := "template/create.html"

	return &TemplateHandler{
		tmplHome:      template.Must(template.ParseFS(templates, append(loggedoutTemplates, homeTemplate)...)),
		tmplSearch:    template.Must(template.ParseFS(templates, append(loggedinTemplates, searchTemplate)...)),
		tmplLogin:     template.Must(template.ParseFS(templates, append(loggedoutTemplates, loginTemplate)...)),
		tmplSignUp:    template.Must(template.ParseFS(templates, append(loggedoutTemplates, signupTemplate)...)),
		tmplDashboard: template.Must(template.ParseFS(templates, append(loggedinTemplates, dashboardTemplate)...)),
		tmplCreate:    template.Must(template.ParseFS(templates, append(loggedinTemplates, createTemplate)...)),
	}
}
