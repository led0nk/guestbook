package templates

import (
	"embed"
	"text/template"
)

type TemplateHandler struct {
	TmplHome      *template.Template
	TmplSearch    *template.Template
	TmplLogin     *template.Template
	TmplSignUp    *template.Template
	TmplDashboard *template.Template
	TmplCreate    *template.Template
}

//go:embed templates/*
var templates embed.FS

func NewTemplateHandler() *TemplateHandler {
	loggedoutTemplates := []string{"templates/index.html", "templates/header.html"}
	loggedinTemplates := []string{"templates/index.html", "templates/loggedinheader.html"}
	homeTemplate := "templates/content.html"
	searchTemplate := "templates/search.html"
	loginTemplate := "templates/login.html"
	signupTemplate := "templates/signup.html"
	dashboardTemplate := "templates/dashboard.html"
	createTemplate := "templates/create.html"

	return &TemplateHandler{
		TmplHome:      template.Must(template.ParseFS(templates, append(loggedoutTemplates, homeTemplate)...)),
		TmplSearch:    template.Must(template.ParseFS(templates, append(loggedinTemplates, searchTemplate)...)),
		TmplLogin:     template.Must(template.ParseFS(templates, append(loggedoutTemplates, loginTemplate)...)),
		TmplSignUp:    template.Must(template.ParseFS(templates, append(loggedoutTemplates, signupTemplate)...)),
		TmplDashboard: template.Must(template.ParseFS(templates, append(loggedinTemplates, dashboardTemplate)...)),
		TmplCreate:    template.Must(template.ParseFS(templates, append(loggedinTemplates, createTemplate)...)),
	}
}
