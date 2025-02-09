package handlers

import (
	"net/http"

	"html/template"

	"github.com/pkg/errors"
)

func render(w http.ResponseWriter, data map[string]interface{}, paths ...string) {
	paths = append(paths, "./templates/main.layout.html")
	tmpl, err := template.ParseFiles(paths...)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.ExecuteTemplate(w, "main", data)
	if err != nil {
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
	}
}
