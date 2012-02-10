/*
 * imgor
 * Copyright (c) 2012 Jimmy Zelinskie
 * Licensed under the MIT license.
 */

package main

import (
	"errors"
  "crypto/sha1"
	"io/ioutil"
	"mime/multipart"
  "fmt"
	"net/http"
	"os"
	"text/template"
)

// Globals
var (
	uploadTemplate *template.Template
	errorTemplate  *template.Template
	imgdir         string
)

// Check for errors
func check(err error) {
	if err != nil {
		panic(err)
	}
}

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
	http.Redirect(w, r, "/view?id="+filename[6:], http.StatusFound)
}

// View Handler
func view(w http.ResponseWriter, r *http.Request) {
	var filename string
	filename = imgdir + r.FormValue("id")

  if filename[len(filename)-3:] == "jpg" {
		w.Header().Set("Content-Type", "image/jpeg")
  } else if filename[len(filename)-3:] == "png" {
		w.Header().Set("Content-Type", "image/png")
	} else {
    panic(errors.New("No supported filetype specified"))
  }

	http.ServeFile(w, r, filename)
}

// Error page's variables
type ErrorPage struct {
	Error error
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
	var err error

	// Set imgdir and make sure it exists!
	imgdir = "./img/"
	_ = os.Mkdir(imgdir[2 : len(imgdir)-1], 0744)

	// Load up templates and check for errors
	uploadTemplate, err = template.ParseFiles("upload.html")
	check(err)
	errorTemplate, err = template.ParseFiles("error.html")
	check(err)

	http.HandleFunc("/", errorHandler(upload))
	http.HandleFunc("/view", errorHandler(view))
	http.ListenAndServe(":3000", nil)
}
