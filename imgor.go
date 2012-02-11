/*
 * imgor
 * Copyright (c) 2012 Jimmy Zelinskie
 * Licensed under the MIT license.
 */

package main

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"text/template"
	"time"
)

// Globals
var (
	uploadTemplate *template.Template
	errorTemplate  *template.Template
	imgdir         string
	templatedir    string
)

// Error page's variables
type ErrorPage struct {
	Error error
}

// Check for errors
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Generate a random filename using a sha1 of the image
func generatefilename(d []byte) string {
	sha := sha1.New()
	return fmt.Sprintf("%x", string(sha.Sum(d))[0:10])
}

// MIME Validator
func validateimage(h *multipart.FileHeader) (ext string, err error) {
	mimeArray := h.Header["Content-Type"]
	mime := mimeArray[0]
	if mime == "image/jpeg" {
		ext = ".jpg"
	} else if mime == "image/png" {
		ext = ".png"
	} else {
		err = errors.New("Unsupported filetype uploaded")
	}
	return
}

// Root Handler
func root(w http.ResponseWriter, r *http.Request) {
	if string(r.URL.Path) == "/styles.css" {
		static(templatedir+"styles.css", w, r)
	} else {
		upload(w, r)
	}
}

// Upload Handler
func upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		// Load the upload page if they aren't posting an image
		uploadTemplate.Execute(w, nil)
		return
	}

	// Get image from POST
	f, h, err := r.FormFile("image")
	defer f.Close()
	check(err)

	// Check MIME and get a file extension
	ext, err := validateimage(h)
	check(err)

	// Read and write the uploaded file to disk
	filebytes, err := ioutil.ReadAll(f)
	check(err)
	filename := imgdir + generatefilename(filebytes) + ext
	err = ioutil.WriteFile(filename, filebytes, 0744)
	check(err)

	// Redirect to the view page
	http.Redirect(w, r, "/view/"+filename[6:], http.StatusFound)
}

// View Handler
func view(w http.ResponseWriter, r *http.Request) {
	filename := string(imgdir + r.URL.Path[len("view/"):])

	if filename[len(filename)-3:] == "jpg" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else if filename[len(filename)-3:] == "png" {
		w.Header().Set("Content-Type", "image/png")
	} else {
		panic(errors.New("No supported filetype specified"))
	}

	// Set expire headers to now + 1 year
	yearlater := time.Now().AddDate(1, 0, 0)
	w.Header().Set("Expires", yearlater.Format(http.TimeFormat))

	http.ServeFile(w, r, filename)
}

// Serve a static file
func static(filename string, w http.ResponseWriter, r *http.Request) {
	// Set MIME
	w.Header().Set("Content-Type", "text/css")

	// Set expire headers to now + 1 year
	yearlater := time.Now().AddDate(1, 0, 0)
	w.Header().Set("Expires", yearlater.Format(http.TimeFormat))

	http.ServeFile(w, r, filename)
}

// One clean error page
func errorHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if e, ok := recover().(error); ok {
				contents := ErrorPage{Error: e}
				w.WriteHeader(http.StatusInternalServerError)
				errorTemplate.Execute(w, contents)
			}
		}()
		fn(w, r)
	}
}

func main() {
	// Set imgdir and make sure it exists!
	imgdir = "./img/"
	_ = os.Mkdir(imgdir[2:len(imgdir)-1], 0744)

	// Load up templates and check for errors
	var err error
	templatedir = "./templates/"
	uploadTemplate, err = template.ParseFiles(templatedir + "upload.html")
	check(err)
	errorTemplate, err = template.ParseFiles(templatedir + "error.html")
	check(err)

	http.HandleFunc("/", errorHandler(root))
	http.HandleFunc("/view/", errorHandler(view))
	http.ListenAndServe(":3000", nil)
}
