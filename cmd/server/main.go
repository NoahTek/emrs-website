package main

import (
	"log"
	"net/http"

	"github.com/NoahTek/emrs-website/internal/database"
	"github.com/NoahTek/emrs-website/internal/handlers"
)

func main() {
	// Connect ot PostgreSQL (postgres://username:password@host:port/dbname)
	dsn := "postgres://emrs_admin:local_password@localhost:5433/emrs_db"

	db, err := database.InitDB(dsn)
	if err != nil {
		log.Fatalf("Cound not initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close db: %v", err)
		}
	}()
	// defer db.Close()

	// Initialize the Handlers Struct with the DB connection
	app := &handlers.Application{
		DB: db,
	}

	// Serve static files (CSS, JS, Uploaded Photos)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	// The Main Dashboard Route (Public Route)
	http.HandleFunc("/", app.HomePage)
	http.HandleFunc("/gallery", app.Photoboard)
	http.HandleFunc("/login", app.HandleLogin)
	http.HandleFunc("/logout", app.HandleLogout)

	// Protected routes wrapped in the RequireAuth Gatekeeper
	// The Upload Route
	http.HandleFunc("/admin/upload", app.RequireAuth(app.UploadPhoto))
	// The HTMX Delete Endpoint
	http.HandleFunc("/admin/photo/", app.RequireAuth(app.DeletePhoto))

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
