package utils

import (
	"html/template"
	log "htmx-blog/logging"
	"net/http"

	"github.com/pkg/errors"
)

func Render(w http.ResponseWriter, data map[string]interface{}, paths ...string) {
	// First parse the layout template
	layoutPath := "./templates/main.layout.html"
	
	// Combine all templates that need to be parsed
	allPaths := append([]string{layoutPath}, paths...)
	
	// Parse all templates at once
	tmpl, err := template.ParseFiles(allPaths...)
	if err != nil {
		log.Error("failed to parse template files: %v", err)
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
		return
	}

	// Execute the main template
	if err := tmpl.ExecuteTemplate(w, "main", data); err != nil {
		log.Error("failed to execute template: %v", err)
		http.Error(w, errors.Wrap(err, "failed to render html page").Error(), http.StatusInternalServerError)
		return
	}
}