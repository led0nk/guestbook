package templates

import (
	"embed"
	"text/template"
)

type TemplateHandler struct {
	TmplHome          *template.Template
	TmplSearch        *template.Template
	TmplSearchResult  *template.Template
	TmplLogin         *template.Template
	TmplForgot        *template.Template
	TmplSignUp        *template.Template
	TmplDashboard     *template.Template
	TmplDashboardUser *template.Template
	TmplCreate        *template.Template
	TmplVerification  *template.Template
	TmplVerMail       *template.Template
	TmplAdmin         *template.Template
	TmplAdminUser     *template.Template
}

//go:embed templates/*
var templates embed.FS

func NewTemplateHandler() *TemplateHandler {
	loggedoutTemplates := []string{"templates/index.html", "templates/header.html"}
	loggedinTemplates := []string{"templates/index.html", "templates/loggedinheader.html"}
	homeTemplate := "templates/content.html"
	searchTemplate := "templates/search.html"
	searchResultTemplate := []string{"templates/searchResult.html"}
	loginTemplate := "templates/auth/login.html"
	forgotTemplate := "templates/auth/forgot.html"
	signupTemplate := "templates/auth/signup.html"
	dashboardTemplate := "templates/user/dashboard.html"
	dashboardUserTemplate := []string{"templates/user/userBlocks.html"}
	createTemplate := "templates/create.html"
	verificationTemplate := "templates/auth/verification.html"
	verMailTemplate := []string{"templates/auth/verMail.html"}
	adminTemplate := "templates/admin/admin.html"
	adminUserTemplate := []string{"templates/admin/adminUserBlocks.html"}

	return &TemplateHandler{
		TmplHome:          template.Must(template.ParseFS(templates, append(loggedoutTemplates, homeTemplate)...)),
		TmplSearch:        template.Must(template.ParseFS(templates, append(loggedinTemplates, searchTemplate)...)),
		TmplSearchResult:  template.Must(template.ParseFS(templates, searchResultTemplate...)),
		TmplLogin:         template.Must(template.ParseFS(templates, append(loggedoutTemplates, loginTemplate)...)),
		TmplForgot:        template.Must(template.ParseFS(templates, append(loggedoutTemplates, forgotTemplate)...)),
		TmplSignUp:        template.Must(template.ParseFS(templates, append(loggedoutTemplates, signupTemplate)...)),
		TmplDashboard:     template.Must(template.ParseFS(templates, append(loggedinTemplates, dashboardTemplate)...)),
		TmplDashboardUser: template.Must(template.ParseFS(templates, dashboardUserTemplate...)),
		TmplCreate:        template.Must(template.ParseFS(templates, append(loggedinTemplates, createTemplate)...)),
		TmplVerification:  template.Must(template.ParseFS(templates, append(loggedoutTemplates, verificationTemplate)...)),
		TmplVerMail:       template.Must(template.ParseFS(templates, verMailTemplate...)),
		TmplAdmin:         template.Must(template.ParseFS(templates, append(loggedinTemplates, adminTemplate)...)),
		TmplAdminUser:     template.Must(template.ParseFS(templates, adminUserTemplate...)),
	}
}
