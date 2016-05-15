package main

import (
	"crypto/rand"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/napsy/httpauth"
	"golang.org/x/crypto/bcrypt"
)

var (
	aaa        httpauth.Authorizer
	userDb     httpauth.GobFileAuthBackend
	bruteCount = 0
	bruteLimit = 10
	authLock   bool
	bruteLock  sync.Mutex
)

func passwordSalt(password string) ([]byte, []byte, error) {
	salt := make([]byte, 8)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, err
	}
	salted := []byte{}
	salted = append(salted, salt...)
	salted = append(salted, []byte(password)...)
	return salt, salted, nil
}

func createUser(username, password, email, role string) error {
	salt, salted, err := passwordSalt(password)
	if err != nil {
		return err
	}
	pwd, err := bcrypt.GenerateFromPassword(salted, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := userDb.SaveUser(httpauth.UserData{Username: username, Hash: pwd, Salt: salt, Role: role}); err != nil {
		return err
	}
	return nil
}

func logout(rw http.ResponseWriter, req *http.Request) {
	if err := aaa.Logout(rw, req); err != nil {
		log.Printf("%v", err)
		return
	}
	http.Redirect(rw, req, "/", http.StatusSeeOther)
}

func login(rw http.ResponseWriter, req *http.Request) {
	if !canAuth() {
		http.Error(rw, "429 Too Many Requests", http.StatusTooManyRequests)
		return
	}
	brutePlus()
	if req.Method == "GET" {
		t, _ := template.ParseFiles("templates/login.html")
		t.Execute(rw, nil)
		return
	}
	username := req.PostFormValue("username")
	password := req.PostFormValue("password")

	if err := aaa.Login(rw, req, username, password, "/"); err != nil && err.Error() == "already authenticated" {
		http.Redirect(rw, req, "/", http.StatusSeeOther)
	} else if err != nil {
		fmt.Println(err)
		http.Redirect(rw, req, "/login", http.StatusSeeOther)
	}
}

func requiresAuth(role string, handleFunc func(w http.ResponseWriter, r *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := aaa.AuthorizeRole(w, r, role, true); err != nil {
			fmt.Println(err)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
		handleFunc(w, r)
	}
}

func brutePlus() {
	bruteLock.Lock()
	defer bruteLock.Unlock()
	bruteCount++
}

func canAuth() bool {
	bruteLock.Lock()
	defer bruteLock.Unlock()
	return !authLock
}

// brute-force password guessing protection
func bruteProtect() {
	for {
		time.Sleep(10 * time.Second)
		bruteLock.Lock()
		if bruteCount >= bruteLimit {
			authLock = true
		} else {
			authLock = false
		}
		bruteCount = 0
		bruteLock.Unlock()
	}
}
func initAuth(adminPwd string) {
	var err error
	userDb, err = httpauth.NewGobFileAuthBackend("users.db")
	if err != nil {
		panic(err)
	}

	if len(adminPwd) > 0 {
		if err := ioutil.WriteFile("users.db", []byte{}, 0666); err != nil {
			log.Printf("Couldn't create 'users.db' file: %v", err)
			os.Exit(1)
		}

		if err := createUser("admin", adminPwd, "root@localhost", "admin"); err != nil {
			log.Printf("Couldn't create admin user: %v", err)
			os.Exit(1)
		}
		log.Printf("Created admin user with password %q ...", adminPwd)
	}
	roles := make(map[string]httpauth.Role)
	roles["user"] = 30
	roles["admin"] = 80
	aaa, err = httpauth.NewAuthorizer(userDb, []byte("cookfwedfie-encryption-key"), "user", roles)
	if err != nil {
		panic(err)
	}

	go bruteProtect()
}
