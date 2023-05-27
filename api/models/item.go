// models/item.go

package models

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Item struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Filename string `json:"filename"`
}

type FileUpload struct {
	File  *multipart.FileHeader `json:"file"`
	Name  string                `json:"name"`
	Price int                   `json:"price"`
}

const (
	host      = "localhost"
	port      = 55000
	user      = "postgres"
	password  = "postgrespw"
	dbname    = "postgres"
	uploadDir = "./uploads"
)

func GetItems(db *sql.DB) ([]Item, error) {
	rows, err := db.Query("SELECT id, name, price, filename FROM items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}

	for rows.Next() {
		var itm Item
		err := rows.Scan(&itm.ID, &itm.Name, &itm.Price, &itm.Filename)
		if err != nil {
			return nil, err
		}

		items = append(items, itm)
	}

	return items, nil
}

var (
	ErrItemNotFound = errors.New("Item not found")
)

func GetItem(db *sql.DB, id int) (Item, error) {
	row := db.QueryRow("SELECT id, name, price, filename FROM items WHERE id = $1", id)

	var itm Item
	err := row.Scan(&itm.ID, &itm.Name, &itm.Price, &itm.Filename)
	if err != nil {
		if err == sql.ErrNoRows {
			return Item{}, ErrItemNotFound
		}
		return Item{}, err
	}

	return itm, nil
}

func AddItem(db *sql.DB, name string, price int, filename string) error {
	_, err := db.Exec("INSERT INTO items (name, price, filename) VALUES ($1, $2, $3)", name, price, filename)
	if err != nil {
		return err
	}

	return nil
}

func UpdateItem(db *sql.DB, id int, name string, price int, r *http.Request) error {
	// Check if a file was uploaded
	file, handler, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		// Retrieve the filename associated with the item from the database
		var filename string
		err = db.QueryRow("SELECT filename FROM items WHERE id = $1", id).Scan(&filename)
		if err != nil {
			if err == sql.ErrNoRows {
				return ErrItemNotFound
			}

			// Delete the associated file from the server
			err = deleteFile(filename)
			if err != nil {
				return err
			}

			// Upload the file and get the generated filename
			filename, err := uploadFile(file, handler)
			if err != nil {
				return err
			}

			// Update the filename in the database
			_, err = db.Exec("UPDATE items SET filename=$1 WHERE id=$2", filename, id)
			if err != nil {
				return err
			}
		}
	}

	_, err = db.Exec("UPDATE items SET name=$1, price=$2 WHERE id=$3", name, price, id)
	if err != nil {
		return err
	}

	return nil
}

func DelItem(db *sql.DB, id int) error {
	var filename string
	err := db.QueryRow("SELECT filename FROM items WHERE id = $1", id).Scan(&filename)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrItemNotFound
		}
		return err
	}

	err = deleteFile(filename)
	if err != nil {
		return err
	}

	_, err = db.Exec("DELETE FROM items WHERE id=$1", id)
	if err != nil {
		return err
	}

	return nil
}

func uploadFile(file multipart.File, handler *multipart.FileHeader) (string, error) {
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano()/int64(time.Millisecond), filepath.Ext(handler.Filename))

	// Save the file to the uploads directory with the generated filename
	filePath := filepath.Join(uploadDir, filename)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Copy the uploaded file's content to the server's file
	_, err = io.Copy(f, file)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func deleteFile(filename string) error {
	filePath := filepath.Join(uploadDir, filename)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func respondWithError(w http.ResponseWriter, statusCode int, message string) {
	respondWithJSON(w, statusCode, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}
