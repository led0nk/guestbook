package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/led0nk/guestbook/db"
	"github.com/led0nk/guestbook/db/jsondb"
	"github.com/led0nk/guestbook/model"
	"golang.org/x/crypto/bcrypt"

	"github.com/google/uuid"
)

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
		//	c.Request.ParseForm()
		email := c.PostForm("email")
		storedpassword, error := u.GetHash(email)
		if error != nil {
			fmt.Println("cannot access right hashpassword", error)
			return
		}
		if err := bcrypt.CompareHashAndPassword(storedpassword, []byte(c.PostForm("password"))); err != nil {
			c.String(http.StatusUnauthorized, "your password/email doesn't match, please try again")
			return
		}
		c.String(http.StatusAccepted, "User authorized")
		//execute xyz
		store := cookie.NewStore([]byte("secret"))

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
