package v1

import (
	"github.com/gorilla/mux"
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
