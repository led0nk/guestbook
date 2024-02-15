package v1

import (
	"errors"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/led0nk/guestbook/cmd/utils"
	"github.com/led0nk/guestbook/internal/database/jsondb"
	"github.com/led0nk/guestbook/internal/model"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) passwordReset() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		newPW := utils.RandomString(8)
		user.Password = []byte(newPW)
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(newPW), 14)
		err = s.mailer.SendPWMail(user, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		user.Password = hashedpassword
		err = s.userstore.UpdateUser(user)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
	}
}

// login authentication and check if user exists
func (s *Server) loginAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		user, error := s.userstore.GetUserByEmail(email)
		if error != nil {
			s.log.Warn("User Error: ", error)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if err := bcrypt.CompareHashAndPassword(user.Password, []byte(r.FormValue("password"))); err != nil {
			s.log.Warn("Hashing Error: ", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		cookie, err := s.tokenstore.CreateToken("session", user.ID, utils.FormValueBool(r.FormValue("Rememberme")))
		if err != nil {
			s.log.Warn("Token Error: ", err)
			return
		}

		http.SetCookie(w, cookie)
		if user.IsAdmin {
			http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
		}
		http.Redirect(w, r, "/user/verify", http.StatusFound)
	}
}

// logoutAuth and deleting session-cookie
func (s *Server) logoutAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "cookie not found", http.StatusBadRequest)
			default:
				s.log.Warn(err)
				http.Error(w, "server error", http.StatusInternalServerError)
			}
		}
		userID, _ := s.tokenstore.GetTokenValue(cookie)
		err = s.tokenstore.DeleteToken(userID)
		if err != nil {
			s.log.Warn("Token Error: ", err)
			return
		}
		cookie.MaxAge = -1
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

// signup authentication and validation of user input
func (s *Server) signupAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		err := jsondb.ValidateUserInput(r.Form)
		if err != nil {
			s.log.Warn("Input Error: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			return
		}
		joinedName := strings.Join([]string{utils.Capitalize(r.FormValue("firstname")), utils.Capitalize(r.FormValue("lastname"))}, " ")
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(r.Form.Get("password")), 14)
		newUser := model.User{
			Email:            html.EscapeString(r.FormValue("email")),
			Name:             html.EscapeString(joinedName),
			Password:         hashedpassword,
			IsAdmin:          false,
			IsVerified:       false,
			VerificationCode: utils.RandomString(6),
			ExpirationTime:   time.Now().Add(time.Minute * 5),
		}
		_, usererr := s.userstore.CreateUser(&newUser)
		if usererr != nil {
			s.log.Warn("User error: ", err)
			http.Redirect(w, r, "/signup", http.StatusFound)
			w.WriteHeader(http.StatusUnauthorized)
		}

		err = s.mailer.SendVerMail(&newUser, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) verifyAuth() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		session, _ := r.Cookie("session")
		userID, err := s.tokenstore.GetTokenValue(session)
		if err != nil {
			s.log.Warn("Token Error: ", err)
			return
		}
		ok, err := s.userstore.CodeValidation(userID, r.FormValue("code"))
		if !ok {
			http.Redirect(w, r, "/user/verify", http.StatusFound)
			s.log.Info("User Error: ", err)
			return
		}
		if err != nil {
			s.log.Error("User Error:", err)
			return
		}
		http.Redirect(w, r, "/user/dashboard", http.StatusFound)
	}
}

func (s *Server) deleteUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		err = s.userstore.DeleteUser(ID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		// http.Redirect(w, r, "/admin/dashboard", http.StatusFound)
	}
}

// TODO: User Template with input Form for editing
func (s *Server) updateUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error", err)
			return
		}
		err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user-update", &user)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

// save updated User data
func (s *Server) saveUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
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
		err = s.userstore.UpdateUser(&updatedUser)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user", &updatedUser)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

func (s *Server) resendVer() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		user.VerificationCode = utils.RandomString(6)
		user.ExpirationTime = time.Now().Add(time.Minute * 5)
		err = s.mailer.SendVerMail(user, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		err = s.userstore.UpdateUser(user)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		err = s.templates.TmplAdminUser.ExecuteTemplate(w, "user", &user)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

func (s *Server) forgotPW() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.userstore.GetUserByEmail(r.FormValue("email"))
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		newPW := utils.RandomString(8)
		user.Password = []byte(newPW)
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(newPW), 14)
		err = s.mailer.SendPWMail(user, s.templates)
		if err != nil {
			s.log.Warn("Mailer Error: ", err)
			return
		}
		user.Password = hashedpassword
		err = s.userstore.UpdateUser(user)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}

func (s *Server) search() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userName := r.URL.Query().Get("name")
		entry, _ := s.bookstore.GetEntryBySnippet(userName)
		err := s.templates.TmplSearchResult.ExecuteTemplate(w, "result", &entry)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}

func (s *Server) submitUserData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, err := uuid.Parse(mux.Vars(r)["ID"])
		if err != nil {
			s.log.Warn("UUID Error: ", err)
			return
		}
		user, err := s.userstore.GetUserByID(userID)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		updatedUser := model.User{
			ID:    user.ID,
			Name:  r.FormValue("Name"),
			Email: r.FormValue("Email"),
		}
		err = s.userstore.UpdateUser(&updatedUser)
		if err != nil {
			s.log.Warn("User Error: ", err)
			return
		}
		err = s.templates.TmplDashboardUser.ExecuteTemplate(w, "user", &updatedUser)
		if err != nil {
			s.log.Warn("Template Error: ", err)
			return
		}
	}
}
