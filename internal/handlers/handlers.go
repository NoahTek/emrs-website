// Package handlers has function handlers
package handlers

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/NoahTek/emrs-website/internal/database"
	"github.com/NoahTek/emrs-website/templates"
)

// Application holds all the dependencies the handlers need
type Application struct {
	DB *sql.DB
}

func (app *Application) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	photos, err := database.GetAllPhotos(app.DB)
	if err != nil {
		log.Printf("Error fetching photos: %v", err)
		http.Error(w, "Could not load gallery", http.StatusInternalServerError)
		return
	}

	// Write a quick HTML shell.
	// (In a full app, you'd make this a BaseLayout.templ component)
	_, _ = w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>EMRS Admin Panel</title>
			<script src="https://unpkg.com/htmx.org@1.9.11"></script>
			<script src="https://cdn.tailwindcss.com"></script>
		</head>
		<body class="bg-gray-900 text-[#ebdbb2] p-8 min-h-screen">
			<div class="max-w-6xl mx-auto border-b border-gray-700 pb-4 mb-4">
				<h1 class="text-3xl font-bold">EMRS Photo Dashboard</h1>
				<p class="text-gray-400">Manage school gallery uploads.</p>
			</div>
	`))

	// Initialize our templ component with data
	gridComponent := templates.GalleryGrid(photos)

	// Render the component directly to the HTTP ResponseWriter
	err = gridComponent.Render(context.Background(), w)
	if err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	// Close the HTML shell
	_, err = w.Write([]byte(`</body></html>`))
	if err != nil {
		log.Printf("failed to clolse HTML shell: %v", err)
	}
}

func (app *Application) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		// Example: extracting ID from URL and deleting it using app.DB
		// id := extractingIDfromURL(r.URL.Path)
		// _ = database.DeletePhoto(app.DB, id)

		// HTMX expects to return the new HTML state.
		// Returning an empty string tells HTMX to swap the target element with nothing (removing it).
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(""))
	}
}
