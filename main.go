package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/led0nk/guestbook/db"
	"github.com/led0nk/guestbook/db/jsondb"
	"github.com/led0nk/guestbook/model"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

var cookies = sessions.NewCookieStore([]byte("secret"))

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")

	var (
		//addr     = flag.String("addr", "localhost:8080", "server port")
		entryStr = flag.String("entrydata", "file://entries.json", "link to entry-database")
		//userStr  = flag.String("userdata", "file://user.json", "link to user-database")
	)
	flag.Parse()
	u, err := url.Parse(*entryStr)
	if err != nil {
		panic(err)
	}
	log.Print(u)
	var gueststore db.GuestBookStorage
	var userstore db.UserStorage
	switch u.Scheme {
	case "file":
		log.Println("opening:", u.Hostname())
		bookStorage, _ := jsondb.CreateBookStorage("./entries.json")
		userStorage, _ := jsondb.CreateUserStorage("./user.json")
		gueststore = bookStorage
		userstore = userStorage
	default:
		panic("bad storage")
	}

	//placeholder
	router.GET("/", handlePage(gueststore))
	router.POST("/submit", submit(gueststore))
	router.POST("/", delete(gueststore))
	router.GET("/login", login())
	router.POST("/login", loginAuth(userstore))
	router.GET("/signup", signup())
	router.POST("/signupauth", signupAuth(userstore))

	error := router.Run("localhost:8080")
	if error != nil {
		log.Fatal(error)
	}
}

// hands over Entries to Handler and prints them out in template
func handlePage(s db.GuestBookStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/content.html")
		searchName := c.Query("q")
		var entries []*model.GuestbookEntry
		if searchName != "" {
			entries, _ = s.GetEntryByName(searchName)
		} else {
			entries, _ = s.ListEntries()
		}

		session, erro := cookies.Get(c.Request, "session")
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   10,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}

		if erro != nil {
			c.String(http.StatusInternalServerError, "internal server error")
			return
		}

		session, erros := cookies.Get(c.Request, "session")
		if erros != nil {
			return
		}
		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   60,
			HttpOnly: true,
		}
		session.Values["user"] = "user[0].ID"

		fmt.Println(session.Values)
		session.Save(c.Request, c.Writer)

		err := tmplt.Execute(c.Writer, &entries)
		if err != nil {
			c.String(http.StatusBadGateway, "error when executing template")
			return
		}

	}
}

// submits guestbook entry (name, message)
func submit(s db.GuestBookStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.ParseForm()
		newEntry := model.GuestbookEntry{Name: c.Request.FormValue("name"), Message: c.Request.FormValue("message")}
		if newEntry.Name == "" {
			return
		}
		s.CreateEntry(&newEntry)
		c.Redirect(http.StatusFound, "/")
	}
}

func delete(s db.GuestBookStorage) gin.HandlerFunc {
	return func(c *gin.Context) {

		c.Request.ParseForm()
		strUuid := c.Request.Form.Get("Delete")
		uuidStr, _ := uuid.Parse(strUuid)

		deleteEntry := model.GuestbookEntry{ID: uuidStr}
		s.DeleteEntry(deleteEntry.ID)
		c.Redirect(http.StatusFound, "/")

	}
}

// show login Form
func login() gin.HandlerFunc {
	return func(c *gin.Context) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/login.html")
		err := tmplt.Execute(c.Writer, nil)
		if err != nil {
			c.String(http.StatusBadGateway, "error when executing template")
			return
		}
	}
}

// show signup Form
func signup() gin.HandlerFunc {
	return func(c *gin.Context) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/signup.html")
		err := tmplt.Execute(c.Writer, nil)
		if err != nil {
			c.String(http.StatusBadGateway, "error when executing template")
			return
		}
	}
}

// login authentication and check if user exists
func loginAuth(u db.UserStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		tmplt, _ := template.ParseFiles("templates/index.html", "templates/header.html", "templates/user.html")
		email := c.PostForm("email")
		user, error := u.GetUserByEmail(email)
		if error != nil {
			fmt.Println("cannot access right hashpassword", error)
			return
		}
		if err := bcrypt.CompareHashAndPassword(user[0].Password, []byte(c.PostForm("password"))); err != nil {
			c.String(http.StatusUnauthorized, "your password/email doesn't match, please try again")
			return
		}

		session, errcookie := cookies.Get(c.Request, "session")

		session.Values["user"] = user[0].ID
		session.Values["name"] = user[0].Name
		session.Save(c.Request, c.Writer)

		if errcookie != nil {
			c.String(http.StatusNotFound, "session not found")
			return
		}
		emptyname, _ := session.Values["name"]
		name, _ := emptyname.(string)
		fmt.Println(name)
		err := tmplt.Execute(c.Writer, name)
		if err != nil {
			c.String(http.StatusBadGateway, "error when executing template")
			return
		}
		// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		// 	"user":       user[0].ID,
		// 	"expiration": time.Now().Add(time.Hour * 24 * 30).Unix(),
		// })

		// tokenString, err := token.SignedString([]byte("top-secret-key"))
		// if err != nil {
		// 	c.String(http.StatusBadRequest, "Failed to create token")
		// 	return
		// }
		// c.SetSameSite(http.SameSiteLaxMode)
		// c.SetCookie("authorization", tokenString, 3600, "/", "", false, true)
		// fmt.Println(c.Cookie("authorization"))
		// c.String(http.StatusOK, tokenString)

	}
}

// signup authentication and validation of user input
func signupAuth(u db.UserStorage) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.ParseForm()
		err := jsondb.ValidateUserInput(c.Request.Form)
		if err != nil {
			fmt.Println("user form not valid:", err)
			c.Redirect(http.StatusBadRequest, "/signup")
			return
		}
		joinedName := strings.Join([]string{c.Request.FormValue("firstname"), c.Request.FormValue("lastname")}, " ")
		hashedpassword, _ := bcrypt.GenerateFromPassword([]byte(c.Request.Form.Get("password")), 14)
		newUser := model.User{Email: c.Request.FormValue("email"), Name: joinedName, Password: hashedpassword}
		u.CreateUser(&newUser)
	}
}
