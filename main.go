package main

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
	"launchpad.net/goamz/aws"
	"launchpad.net/goamz/s3"
)

var (
	templates map[string]*template.Template
	bucket    *s3.Bucket
)

func checkFor500s(err error) {
	if err != nil {
		panic(err)
	}
}

func handlerFor500s(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err, ok := recover().(error); ok {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}()
		fn(w, r)
	}
}

// uniqueImageName generates a 10 digit hex sha1 of a []byte
func uniqueImageName(image []byte) (string, error) {
	sha := sha1.New()
	_, err := sha.Write(image)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha.Sum(nil)[:10]), nil
}

// validateImage reads an image's headers to determine the type of image.
func validateImage(image []byte) (mimetype, extension string, err error) {
	if bytes.Equal(image[:2], []byte{0xff, 0xd8}) {
		return "image/jpeg", "jpg", nil
	}
	if bytes.Equal(image[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		return "image/png", "png", nil
	}
	if bytes.Equal(image[:6], []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}) ||
		bytes.Equal(image[:6], []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}) {
		return "image/gif", "gif", nil
	}
	return "", "", errors.New("Unaccepted content type")
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := templates["home"].ExecuteTemplate(w, "base", nil)
	checkFor500s(err)
}

func ImageUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "HTTP method is not POST", http.StatusNotFound)
	}

	f, _, err := r.FormFile("image")
	checkFor500s(err)
	defer f.Close()
	rawImage, err := ioutil.ReadAll(f)
	checkFor500s(err)
	mime, ext, err := validateImage(rawImage)
	checkFor500s(err)
	name, err := uniqueImageName(rawImage)
	checkFor500s(err)

	err = bucket.Put(name, rawImage, mime, s3.ACL("bucket-owner-full-control"))
	checkFor500s(err)

	http.Redirect(w, r, "/"+name+"."+ext, http.StatusFound)
}

func ImageHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	i := strings.Split(vars["image"], ".")
	image, err := bucket.Get(i[0])
	if err != nil {
		http.Error(w, "Unable to find file", http.StatusNotFound)
	}
	_, err = w.Write(image)
	checkFor500s(err)
	mime, _, err := validateImage(image)
	checkFor500s(err)
	w.Header().Set("Content-Type", mime)
}

func StaticHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortPath := vars["path"]
	path := "static/" + shortPath
	fileinfo, err := os.Stat(path)
	if err != nil || fileinfo.IsDir() {
		http.Error(w, "Unable to find file", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, path)
}

func init() {
	// Templates
	templates = make(map[string]*template.Template)
	templates["home"] = template.Must(template.ParseFiles("templates/home.html", "templates/base.html"))
	templates["404"] = template.Must(template.ParseFiles("templates/404.html", "templates/base.html"))

	// S3 Auth
	sss := s3.New(aws.Auth{os.Getenv("AWS_ACCESS_KEY"), os.Getenv("AWS_SECRET_KEY")}, aws.USEast)
	bucket = sss.Bucket("imgor")
}

func main() {
	r := mux.NewRouter()
	s := r.PathPrefix("/static").Subrouter()
	s.HandleFunc("/{path:(.*?)}", handlerFor500s(StaticHandler)).Name("static")
	r.HandleFunc("/upload", handlerFor500s(ImageUploadHandler)).Name("image")
	r.HandleFunc("/", handlerFor500s(HomeHandler)).Name("home")
	r.HandleFunc("/{image}", handlerFor500s(ImageHandler)).Name("image")

	http.Handle("/", r)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}
