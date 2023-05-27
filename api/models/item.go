// models/item.go

package models

import (
	"database/sql"
	"errors"
	"mime/multipart"
	"net/http"
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

func UpdateItem(db *sql.DB, id int, name string, price int, r *http.Request, uploadFile func(file multipart.File, handler *multipart.FileHeader) (string, error), deleteFile func(filename string) error) error {
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

func DelItem(db *sql.DB, id int) (string, error) {
	var filename string
	err := db.QueryRow("SELECT filename FROM items WHERE id = $1", id).Scan(&filename)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrItemNotFound
		}
		return "", err
	}

	_, err = db.Exec("DELETE FROM items WHERE id=$1", id)
	if err != nil {
		return "", err
	}

	return filename, nil
}
