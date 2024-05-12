package v1

import (
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/led0nk/guestbook/cmd/utils"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/model"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) passwordReset(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.passwordReset")
	defer span.End()

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}
	newPW := utils.RandomString(8)
	user.Password = []byte(newPW)
	hashedpassword, err := bcrypt.GenerateFromPassword([]byte(newPW), 14)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to gernerate hashed password", "error", err)
		return
	}
	err = s.mailer.SendPWMail(user, s.templates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to send password-mail", "error", err)
		return
	}
	user.Password = hashedpassword
	err = s.userstore.UpdateUser(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to update user", "error", err)
		return
	}
}

// login authentication and check if user exists
func (s *Server) loginAuth(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.loginAuth")
	defer span.End()

	email := r.FormValue("email")
	user, err := s.userstore.GetUserByEmail(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to compare passwords", "error", err)
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	cookie, err := s.tokenstore.CreateToken(ctx, "session", s.domain, user.ID, utils.FormValueBool(r.FormValue("Rememberme")))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to create token", "error", err)
		return
	}

	http.SetCookie(w, cookie)
	if user.IsAdmin {
		http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	}
	http.Redirect(w, r, "/user/verify", http.StatusFound)
}

// logoutAuth and deleting session-cookie
func (s *Server) logoutAuth(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.logoutAuth")
	defer span.End()

	cookie, err := r.Cookie("session")
	if err != nil {
		switch {
		case errors.Is(err, http.ErrNoCookie):
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			s.log.ErrorContext(ctx, "failed to get cookie", "error", err)
			http.Error(w, "cookie not found", http.StatusBadRequest)
		default:
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			s.log.ErrorContext(ctx, "error while getting cookie", "error", err)
			http.Error(w, "server error", http.StatusInternalServerError)
		}
	}
	userID, err := s.tokenstore.GetTokenValue(ctx, cookie)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get token value", "error", err)
		return
	}
	err = s.tokenstore.DeleteToken(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to delete token", "error", err)
		return
	}
	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/login", http.StatusFound)
}

// signup authentication and validation of user input
func (s *Server) signupAuth(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.signupAuth")
	defer span.End()

	err := r.ParseForm()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse form", "error", err)
		return
	}
	err = jsondb.ValidateUserInput(r.Form)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to validate user input", "error", err)
		http.Redirect(w, r, "/signup", http.StatusFound)
		return
	}
	joinedName := strings.Join([]string{utils.Capitalize(r.FormValue("firstname")), utils.Capitalize(r.FormValue("lastname"))}, " ")
	hashedpassword, err := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), 14)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to generate password", "error", err)
		return
	}
	newUser := model.User{
		Email:            html.EscapeString(r.FormValue("email")),
		Name:             html.EscapeString(joinedName),
		Password:         hashedpassword,
		IsAdmin:          false,
		IsVerified:       false,
		VerificationCode: utils.RandomString(6),
		ExpirationTime:   time.Now().Add(time.Minute * 5),
	}
	_, err = s.userstore.CreateUser(ctx, &newUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to create user", "error", err)
		http.Redirect(w, r, "/signup", http.StatusFound)
		w.WriteHeader(http.StatusUnauthorized)
	}

	err = s.mailer.SendVerMail(&newUser, s.domain, s.templates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to send verification mail", "error", err)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) verifyAuth(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.verifyAuth")
	defer span.End()

	err := r.ParseForm()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse form", "error", err)
		return
	}
	session, err := r.Cookie("session")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to find cookie", "error", err)
		return
	}
	userID, err := s.tokenstore.GetTokenValue(ctx, session)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get token value", "error", err)
		return
	}
	ok, err := s.userstore.CodeValidation(ctx, userID, r.FormValue("code"))
	if !ok {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		http.Redirect(w, r, "/user/verify", http.StatusFound)
		s.log.ErrorContext(ctx, "verification code is not matching", "error", err)
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to validate verification code", "error", err)
		return
	}
	http.Redirect(w, r, "/user/dashboard", http.StatusFound)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.deleteUser")
	defer span.End()

	ID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	err = s.userstore.DeleteUser(ctx, ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to delete user", "error", err)
		return
	}
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.updateUser")
	defer span.End()

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}
	err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user-update", &user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", "error", err)
		return
	}
}

// save updated User data
func (s *Server) saveUser(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.saveUser")
	defer span.End()

	err := r.ParseForm()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse form", "error", err)
		return
	}

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}

	updatedUser := model.User{
		ID:               user.ID,
		Email:            r.FormValue("Email"),
		Name:             user.Name,
		Password:         user.Password,
		IsAdmin:          utils.FormValueBool(r.FormValue("Admin")),
		IsVerified:       utils.FormValueBool(r.FormValue("Verified")),
		VerificationCode: user.VerificationCode,
		ExpirationTime:   user.ExpirationTime,
	}
	err = s.userstore.UpdateUser(ctx, &updatedUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to update user", "error", err)
		return
	}
	err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user", &updatedUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", "error", err)
		return
	}
}

func (s *Server) resendVer(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.resendVer")
	defer span.End()

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}
	user.VerificationCode = utils.RandomString(6)
	user.ExpirationTime = time.Now().Add(time.Minute * 5)
	err = s.mailer.SendVerMail(user, s.domain, s.templates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to send verification mail", "error", err)
		return
	}
	err = s.userstore.UpdateUser(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to update user", "error", err)
		return
	}
	err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user", &user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", "error", err)
		return
	}
}

func (s *Server) forgotPW(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.resendVer")
	defer span.End()

	user, err := s.userstore.GetUserByEmail(ctx, r.FormValue("email"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}
	newPW := utils.RandomString(8)
	user.Password = []byte(newPW)
	hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(newPW), 14)
	err = s.mailer.SendPWMail(user, s.templates)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to generate password", "error", err)
		return
	}
	user.Password = hashedpassword
	err = s.userstore.UpdateUser(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to update user", "error", err)
		return
	}
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.search")
	defer span.End()

	userName := r.URL.Query().Get("name")
	entry, err := s.bookstore.GetEntryBySnippet(ctx, userName)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get entry", "error", err)
		return
	}
	err = s.templates.TmplSearchResult.ExecuteTemplate(w, "result", &entry)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", "error", err)
		return
	}
}

func (s *Server) submitUserData(w http.ResponseWriter, r *http.Request) {
	var span trace.Span
	ctx := r.Context()
	ctx, span = tracer.Start(ctx, "server.submitUserData")
	defer span.End()

	userID, err := uuid.Parse(r.PathValue("ID"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to parse uuid", "error", err)
		return
	}
	user, err := s.userstore.GetUserByID(ctx, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to get user", "error", err)
		return
	}
	updatedUser := model.User{
		ID:               user.ID,
		Password:         user.Password,
		Name:             html.EscapeString(r.FormValue("Name")),
		Email:            html.EscapeString(r.FormValue("Email")),
		IsAdmin:          user.IsAdmin,
		IsVerified:       user.IsVerified,
		VerificationCode: user.VerificationCode,
		ExpirationTime:   user.ExpirationTime,
	}
	err = s.userstore.UpdateUser(ctx, &updatedUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to update user", "error", err)
		return
	}
	err = s.templates.TmplDashboardUser.ExecuteTemplate(w, "user", &updatedUser)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		s.log.ErrorContext(ctx, "failed to execute template", "error", err)
		return
	}
}
