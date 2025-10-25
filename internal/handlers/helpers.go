package handlers

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
)

func handleError(w http.ResponseWriter, log *log.Logger, err error, message string, status int) {
	log.Printf("%s: %v", message, err)
	http.Error(w, message, status)
}

func renderError(w http.ResponseWriter, code int, title, message, projectRoot string) {
	w.WriteHeader(code)
	tmpl, err := template.ParseFiles(filepath.Join(projectRoot, "static", "error.html"))
	if err != nil {
		http.Error(w, message, code)
		return
	}
	data := map[string]interface{}{
		"Code":    code,
		"Title":   title,
		"Message": message,
	}
	tmpl.Execute(w, data)
}
