/*
 * The MIT License (MIT)
 *
 * Copyright (c) 2016 Luka Napotnik <luka.napotnik@gmail.com>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/http/fcgi"
	"strings"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := aaa.CurrentUser(w, r)
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	t.Execute(w, user.Username) // merge.
}

func editHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := aaa.CurrentUser(w, r)
	if r.Method == "POST" {
		name := r.Referer()
		idx := strings.LastIndex(name, "/")
		contents := r.PostFormValue("contents")
		if err := savePage(name[idx:], contents); err != nil {
			panic(err)
		}
		http.Redirect(w, r, fmt.Sprintf("/page/%s", name[idx:]), http.StatusSeeOther)
		return
	}
	path := r.URL.EscapedPath()
	filename := ""
	if idx := strings.LastIndex(path, "/"); idx == -1 {
		return
	} else {
		filename = path[idx+1:]
	}

	p, meta, err := getPageSource(filename)
	if err != nil && user.Role != "admin" {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	// Load template
	t, _ := template.ParseFiles("templates/edit.html")
	t.Execute(w, pageSource{getTitle(filename), meta.ImageURL, string(p)})
}

func pageHandler(w http.ResponseWriter, r *http.Request) {
	user, _ := aaa.CurrentUser(w, r)
	path := r.URL.EscapedPath()
	filename := ""
	if idx := strings.LastIndex(path, "/"); idx == -1 {
		return
	} else {
		filename = path[idx+1:]
	}

	p, _, err := getPage(filename)
	if err != nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	p.User = user.Username
	// Load template
	t, _ := template.ParseFiles("templates/page.html")
	t.Execute(w, p)
}

func main() {
	var (
		useCGI  = flag.Bool("cgi", false, "use CGI server")
		port    = flag.Int("p", 8000, "port to use")
		initArg = flag.String("init", "", "initialize with given admin password")
		mux     = http.NewServeMux()
	)
	flag.Parse()

	initAuth(*initArg)

	mux.HandleFunc("/page/", pageHandler)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/logout", logout)

	apps := newWebApps()
	// Middleware handler
	mux.Handle("/app/", apps)

	mux.HandleFunc("/edit/", requiresAuth("admin", editHandler))
	mux.HandleFunc("/", indexHandler)
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	if *useCGI {
		listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
		if err != nil {
			log.Fatal(err)
		}
		defer listener.Close()
		log.Printf("%v", fcgi.Serve(listener, mux))
		return
	}
	log.Printf("%v", http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *port), mux))
}
