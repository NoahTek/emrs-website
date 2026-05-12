// Package handlers has function handlers
package handlers

import (
	"context"
	"database/sql"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/NoahTek/emrs-website/internal/database"
	"github.com/NoahTek/emrs-website/templates"
)

// Application holds all the dependencies the handlers need
type Application struct {
	DB *sql.DB
}

func (app *Application) Dashboard(w http.ResponseWriter, r *http.Request) {
	isAdmin := app.isAuthenticated(r)

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
			<title>EMRS Photo Dashboard</title>
			<script src="https://unpkg.com/htmx.org@1.9.11"></script>
			<script src="https://cdn.tailwindcss.com"></script>
		</head>
		<body class="bg-gray-900 text-[#ebdbb2] p-8 min-h-screen">
			<div class="max-w-6xl mx-auto border-b border-gray-700 pb-4 mb-4">
				<h1 class="text-3xl font-bold">EMRS Photo Dashboard</h1>
				<p class="text-gray-400">Manage school gallery uploads.</p>
			</div>
	`))

	// Add Logout button or Login link based on status
	if isAdmin {
		_, _ = w.Write([]byte(`<form action="/logout" method="POST"><button class="text-sm text-gray-400 hover:text-white">Logout</button></form>`))
	} else {
		_, _ = w.Write([]byte(`<a href="/login" class="text-gray-400 hover:text-white">Admin Login</a>`))
	}
	_, _ = w.Write([]byte(`</div>`))

	// ONLY render the upload form if they are an admin
	if isAdmin {
		// Rendering the templ components using the REAL data
		uploadForm := templates.UploadForm()
		_ = uploadForm.Render(context.Background(), w)
	}

	// Initialize our templ component with data
	gridComponent := templates.GalleryGrid(photos, isAdmin)

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
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Extract the ID from the URL path
	pathParts := strings.Split(r.URL.Path, "/")
	idStr := pathParts[len(pathParts)-1]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID fromat", http.StatusBadRequest)
		return
	}

	// 2. Delete the record from PostgreSQL
	imageURL, err := database.DeletePhoto(app.DB, id)
	if err != nil {
		log.Printf("Error deleting photo from DB: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// 3. Delete the physical file from the hard drive
	localFilePath := strings.TrimPrefix(imageURL, "/")

	err = os.Remove(localFilePath)
	if err != nil {
		// log the error but don't crash the request, since the DB record is already gone
		log.Printf("Warning: Failed to delete physical file %s: %v", localFilePath, err)
	} else {
		log.Printf("Successfully deleted physical file: %s", localFilePath)
	}

	// 4. Return an empty 200 OK respose
	w.WriteHeader(http.StatusOK)
}

// UploadPhoto handles the multipart form submission
func (app *Application) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse the multipart form (limiting memory to 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "File too large", http.StatusBadRequest)
		return
	}

	// 2. Extract the title string and the actual payload
	title := r.FormValue("title")
	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 3. Save the file locally to the static/uploads directory
	fileName := header.Filename
	savePath := filepath.Join("static", "uploads", fileName)

	dst, err := os.Create(savePath)
	if err != nil {
		log.Printf("Error saving file: %v", err)
		http.Error(w, "failed to save file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()
	_, _ = io.Copy(dst, file)

	// 4. Save the record to PostgreSQL
	imageURL := "/static/uploads/" + fileName
	id, err := database.InsertPhoto(app.DB, title, imageURL)
	if err != nil {
		log.Printf("Error inserting to DB: %v", err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	// 5. Render only the newly created Gallery Card
	newItem := database.PhotoRecord{
		ID:       id,
		Title:    title,
		ImageURL: imageURL,
	}

	card := templates.GalleryCard(newItem, true)
	_ = card.Render(r.Context(), w)
}

// SECURITY & AUTHENTICATION

// isAuthenticated is a helper to check if the user has a valid admin cookie
func (app *Application) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("emrs_session")
	if err != nil {
		return false
	}
	return cookie.Value == "admin_authorized"
}

// RequireAuth is a MiddleWare that wraps around our protected routes.
// If the user doesn't have the cookie, it rejects the request entirely.
func (app *Application) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.isAuthenticated(r) {
			http.Error(w, "Unauthorized. Admin access required.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// HandleLogin processes the login form
func (app *Application) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Render the login HTML page
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>login</title><script src="https://cdn.tailwindcss.com"></script></head><body class="bg-gray-900">`))
		_ = templates.LoginForm().Render(r.Context(), w)
		_, _ = w.Write([]byte(`</body></html>`))
		return
	}

	// If it's a POST request, check the credentials
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Hardcoded for now. (Move to Postgres later!)
	if username == "admin" && password == "emrs2026" {
		cookie := &http.Cookie{
			Name:     "emrs_session",
			Value:    "admin_authorized",
			Path:     "/",
			HttpOnly: true, // Security best practice: Javascript cannot read this cookie
			MaxAge:   3600, // Expires in 1 hour
		}
		http.SetCookie(w, cookie)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Faild Login
	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

// HandleLogout destroys the cookie
func (app *Application) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:   "emrs_session",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // Deltes the cookie instantly
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
