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
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/russross/blackfriday"
)

type cachedPage struct {
	meta       pageMeta
	p          page
	lastAccess time.Time
}

var (
	pageCache map[string]*cachedPage
	cacheLock sync.Mutex
)

const pageMetaSeparator = "---"

type pageMeta struct {
	Subtitle string
	ImageURL string
}

type pageSource struct {
	Title      string
	TitleImage string
	Contents   string
}
type page struct {
	Title      string
	Subtitle   string
	TitleImage string
	User       string
	PageName   string
	Contents   template.HTML
}

func savePage(title string, contents string) error {
	filename := fmt.Sprintf("page/%s.md", title)
	return ioutil.WriteFile(filename, []byte(contents), 0666)
}

func loadPage(filename string) (page, pageMeta, error) {
	ctx, err := ioutil.ReadFile(fmt.Sprintf("page/%s.md", filename))
	if err != nil {
		return page{}, pageMeta{}, err
	}

	// Read page metadata
	meta, idx, err := readPageMeta(string(ctx))
	// Now decode markdown into HTML
	ctx = blackfriday.MarkdownCommon(ctx[idx:])
	// Create data for the HTML template
	p := page{
		Title:      getTitle(filename),
		Subtitle:   meta.Subtitle,
		TitleImage: meta.ImageURL,
		PageName:   filename,
		Contents:   template.HTML(string(ctx)),
	}
	return p, meta, nil
}

func getPageSource(filename string) ([]byte, pageMeta, error) {

	ctx, err := ioutil.ReadFile(fmt.Sprintf("page/%s.md", filename))
	if err != nil {
		return nil, pageMeta{}, err
	}
	meta, _, err := readPageMeta(string(ctx))
	if err != nil {
		return nil, pageMeta{}, err
	}
	return ctx, meta, nil
}
func getPage(name string) (page, pageMeta, error) {
	cacheLock.Lock()
	defer cacheLock.Unlock()
	cached, ok := pageCache[name]
	if !ok {
		p, meta, err := loadPage(name)
		if err != nil {
			return p, meta, err
		}
		log.Printf("Creating and caching page %q", getTitle(name))
		cached = &cachedPage{p: p, meta: meta, lastAccess: time.Now()}
		pageCache[name] = cached
	} else {
		// Check if file changed since cached.
		stat, err := os.Stat(fmt.Sprintf("page/%s.md", name))
		if err != nil {
			log.Printf("Removing page %q from cache: couldn't stat", getTitle(name))
			delete(pageCache, name)
			return page{}, pageMeta{}, errors.New("page removed")
		}
		if stat.ModTime().Sub(cached.lastAccess) > 0 {
			log.Printf("Page %q was modified, updating cache", getTitle(name))
			// Update page
			p, meta, err := loadPage(name)
			if err != nil {
				return p, meta, err
			}
			cached = &cachedPage{p: p, meta: meta, lastAccess: time.Now()}
			pageCache[name] = cached
		}
		log.Printf("Reading page %q from cache", getTitle(name))
	}
	return cached.p, cached.meta, nil
}

func getTitle(page string) string {
	return strings.Replace(page, "-", " ", -1)
}

func readPageMeta(ctx string) (pageMeta, int, error) {
	meta := pageMeta{}
	idx := strings.Index(ctx, pageMetaSeparator)
	if idx == -1 {
		return meta, 0, nil
	}

	if _, err := toml.Decode(ctx[:idx], &meta); err != nil {
		return meta, 0, err
	}
	return meta, idx + 3, nil
}

func init() {
	pageCache = make(map[string]*cachedPage)
}
