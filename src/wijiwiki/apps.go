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
	"errors"
	"log"
	"net/http"
	"strings"
)

type webApps struct {
	apps map[string]http.Handler
}

func newWebApps() *webApps {
	return &webApps{apps: make(map[string]http.Handler)}
}

func (apps *webApps) Add(name string, handler http.Handler) {
	apps.apps[name] = handler
}
func (apps webApps) getAppName(path string) string {
	appName := ""
	if idx := strings.Index(path, "/"); idx == -1 {
		return ""
	} else {
		appName = path[idx+1:]
	}
	return appName
}

func (apps *webApps) Proxy(w http.ResponseWriter, r *http.Request, app string) error {
	application, ok := apps.apps[app]
	if !ok {
		return errors.New("application not installed")
	}
	application.ServeHTTP(w, r)
	return nil
}

func (apps webApps) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	app := apps.getAppName(r.URL.EscapedPath()[1:])
	log.Printf("Accessing app %q", app)
	if err := apps.Proxy(w, r, app); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
}
