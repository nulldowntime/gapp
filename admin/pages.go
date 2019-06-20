package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//Page struct for pages
type Page struct {
	URL     string
	Logo    string
	Name    string
	View    string
	Contact template.HTML
}

// Controller handle other requests
var Controller = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	uri := r.RequestURI

	var p Page

	switch uri {
	case "/":
		p = Page{Name: "Login", View: "login"}

		if r.Method == "POST" {

			fmt.Println(r.FormValue("email"))
			fmt.Println(r.FormValue("password"))

		}

		break
	case "/register":
		p = Page{Name: "Register", View: "register"}

		if r.Method == "POST" {

			// Parse our multipart form, 10 << 20 specifies a maximum
			// upload of 10 MB files.
			r.ParseMultipartForm(10 << 20)
			// FormFile returns the first file for the given key `myFile`
			// it also returns the FileHeader so we can get the Filename,
			// the Header and the size of the file
			file, _, err := r.FormFile("credentials")
			if err != nil {
				fmt.Println("Error Retrieving the File")
				fmt.Println(err)
				return
			}
			defer file.Close()
			/*
				fmt.Printf("Uploaded File: %+v\n", handler.Filename)
				fmt.Printf("File Size: %+v\n", handler.Size)
				fmt.Printf("MIME Header: %+v\n", handler.Header)
			*/
			// read all of the contents of our uploaded file into a
			// byte array
			fileBytes, err := ioutil.ReadAll(file)
			if err != nil {
				fmt.Println(err)
			}

			u := User{
				Email:       r.FormValue("email"),
				Password:    r.FormValue("password"),
				Credentials: fileBytes,
			}

			// If modifying these scopes, delete your previously saved token.json.
			config, err := google.ConfigFromJSON(u.Credentials, gmail.GmailReadonlyScope)
			if err != nil {
				log.Fatalf("Unable to parse client secret file to config: %v", err)
			}

			config.RedirectURL = os.Getenv("URL") + "auth/" + u.Email + "/"

			u.Config = config

			checkUser := GetUserByEmail(u.Email)
			if checkUser.ID.Hex() != "" {

				UpdateUser(checkUser.ID.Hex(), u)

			} else {

				CreateUser(u)

			}

			authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)

			http.Redirect(w, r, authURL, 301)

		}

		break

	}

	p.URL = os.Getenv("URL")

	parsedTemplate, err := template.ParseFiles(
		"template/index.html",
		"template/views/"+p.View+".html",
	)

	if err != nil {
		log.Println("Error ParseFiles:", err)
		return
	}

	err = parsedTemplate.Execute(w, p)

	if err != nil {
		log.Println("Error Execute:", err)
		return
	}

})

// AuthController handle auth requests
var AuthController = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	// URL vars
	vars := mux.Vars(r)

	email := vars["email"]
	code := r.FormValue("code")

	u := GetUserByEmail(email)

	tok, err := u.Config.Exchange(context.TODO(), code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	u.Token = tok

	UpdateUser(u.ID.Hex(), u)

	http.Redirect(w, r, os.Getenv("URL")+"/login", 301)

})

//StartApp start app on specify port
func StartApp() {

	muxRouter := mux.NewRouter().StrictSlash(true)

	muxRouter.Handle("/login", Controller).Methods("GET", "POST")
	muxRouter.Handle("/register", Controller).Methods("GET", "POST")
	muxRouter.Handle("/auth/{email}", AuthController).Methods("GET", "POST")

	// add static file prefix
	muxRouter.PathPrefix("/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("static/"))))

	err := http.ListenAndServe(":"+os.Getenv("APP_PORT"), handlers.CompressHandler(muxRouter))
	if err != nil {
		log.Fatal("error starting http server :: ", err)
		return
	}

}
