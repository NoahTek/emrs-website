// Package database is database
package database

import (
	"database/sql"
	"fmt"
	"log"

	// Importing the pgx driver anonymously so database/sql can use it.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// InitDB connects to Postgres and ensures schema exists
func InitDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Pinging to ensure the connection is valid.
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgresSQL!")

	// Initializing tables.
	if err := createSchema(db); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return db, nil
}

func createSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS gallery_items (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		image_url TEXT NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}

	log.Println("Database schema verified.")
	return nil
}

// PhotoRecord represents a single row from the gallery_items table
type PhotoRecord struct {
	ID       int
	Title    string
	ImageURL string
}

// GetAllPhotos retrieves all images sorted by newest first
func GetAllPhotos(db *sql.DB) ([]PhotoRecord, error) {
	query := `SELECT id, title, image_url FROM gallery_items ORDER BY created_at DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("failed to close rows: %v", err)
		}
	}()
	// defer rows.Close() // Ensuring the database connection is released.

	var photos []PhotoRecord
	for rows.Next() {
		var p PhotoRecord
		if err := rows.Scan(&p.ID, &p.Title, &p.ImageURL); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}

	// Checking for any errors encountered during iteration
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return photos, nil
}

// InsertPhoto adds a new image record to the database
func InsertPhoto(db *sql.DB, title, imageURL string) error {
	// Notice: Using $1 and $2 for Postgres parameterized queries to prevent SQL injection
	query := `INSERT INTO gallery_items (title,image_url) VALUES ($1, $2)`
	_, err := db.Exec(query, title, imageURL)
	return err
}

// DeletePhoto removes a record by its ID
func DeletePhoto(db *sql.DB, id int) error {
	query := `DELETE FROM gallery_items WHERE id = $1`
	_, err := db.Exec(query, id)
	return err
}
