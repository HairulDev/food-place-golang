// handlers/item.go

package handlers

import (
	"encoding/json"
	"inventory/api/models"
	"inventory/database"
	"io"
	"net/http"
	"strconv"

	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	host      = "localhost"
	port      = 55000
	user      = "postgres"
	password  = "postgrespw"
	dbname    = "postgres"
	uploadDir = "./uploads"
)

type Item struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Filename string `json:"filename"`
}

func GetItems(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	items, err := models.GetItems(db)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, items)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	item, err := models.GetItem(db, id)
	if err != nil {
		if err == models.ErrItemNotFound {
			respondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, item)
}

func AddItem(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid price")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to read file")
		return
	}
	defer file.Close()

	filename, err := uploadFile(file, handler)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	db := database.GetDBConnection()
	defer db.Close()

	err = models.AddItem(db, name, price, filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Add item successfully"})
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	// Get the item data from the form values
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid price")
		return
	}

	err = models.UpdateItem(db, id, name, price, r)
	if err != nil {
		if err == models.ErrItemNotFound {
			respondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "Update item successfully"})
}

func DelItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	err = models.DelItem(db, id)
	if err != nil {
		if err == models.ErrItemNotFound {
			respondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
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
