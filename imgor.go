package main

import (
	"errors"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"text/template"
)

// Check for errors
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Page Templates
var (
	uploadTemplate *template.Template
	errorTemplate  *template.Template
)

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
	check(err)
	defer f.Close()

	// Check MIME and get a file extension
	_, err = validateimage(h)
	check(err)

	// Read and write the uploaded file to disk
	filename := "./img/" + h.Filename
	filebytes, err := ioutil.ReadAll(f)
	check(err)
	err = ioutil.WriteFile(filename, filebytes, 0744)
	check(err)

	// Redirect to the view page
	http.Redirect(w, r, "/view?id="+filename[6:], http.StatusFound)
}

// View Handler
func view(w http.ResponseWriter, r *http.Request) {
	var filename string
	filename = "./img/" + r.FormValue("id")

	if filename[len(filename)-4:] == ".jpg" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else if filename[len(filename)-4:] == ".png" {
		w.Header().Set("Content-Type", "image/png")
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

	// Load up templates and check for errors
	uploadTemplate, err = template.ParseFiles("upload.html")
	check(err)
	errorTemplate, err = template.ParseFiles("error.html")
	check(err)

	http.HandleFunc("/", errorHandler(upload))
	http.HandleFunc("/view", errorHandler(view))
	http.ListenAndServe(":3000", nil)
}
