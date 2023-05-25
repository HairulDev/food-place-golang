package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
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

type FileUpload struct {
	File  *multipart.FileHeader `json:"file"`
	Name  string                `json:"name"`
	Price int                   `json:"price"`
}

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/item", getItems).Methods("GET")
	router.HandleFunc("/item/{id}", getItem).Methods("GET")
	router.HandleFunc("/item", addItem).Methods("POST")
	router.HandleFunc("/item/{id}", updateItem).Methods("PUT")
	router.HandleFunc("/item/{id}", deleteItem).Methods("DELETE")

	// Buat handler untuk serve file dari folder "uploads"
	uploadsHandler := http.FileServer(http.Dir(uploadDir))

	// Tambahkan route untuk handler uploadsHandler
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsHandler))

	// Buat folder upload jika belum ada
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	// Buat string koneksi untuk database PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Membuka koneksi ke database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Menguji koneksi ke database
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	// Buat instance CORS dengan konfigurasi default
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // Atur sesuai kebutuhan Anda
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	handler := c.Handler(router)

	var port = 8000
	fmt.Println("Connected to: " + strconv.Itoa(port))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), handler))

}

func getItems(w http.ResponseWriter, r *http.Request) {
	db := getDBConnection()
	defer db.Close()

	rows, err := db.Query("SELECT id, name, price, filename FROM items")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []Item{}

	for rows.Next() {
		var itm Item
		err := rows.Scan(&itm.ID, &itm.Name, &itm.Price, &itm.Filename)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		items = append(items, itm)
	}

	respondWithJSON(w, http.StatusOK, items)
}

func getItem(w http.ResponseWriter, r *http.Request) {
	db := getDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	row := db.QueryRow("SELECT id, name, price, filename FROM items WHERE id = $1", id)

	var itm Item
	err = row.Scan(&itm.ID, &itm.Name, &itm.Price, &itm.Filename)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, itm)
}

func addItem(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid price")
		return
	}

	// Parse the multipart form to access the file
	err = r.ParseMultipartForm(10 << 20) // Maximum file size of 10 MB
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get the file from the form
	file, handler, err := r.FormFile("file")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to read file")
		return
	}
	defer file.Close()

	// Upload the file and get the generated filename
	filename, err := uploadFile(file, handler)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Insert the item data and generated filename into the database
	db := getDBConnection()
	defer db.Close()

	_, err = db.Exec("INSERT INTO items (name, price, filename) VALUES ($1, $2, $3)", name, price, filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"message": "Add item successfully"})
}

func updateItem(w http.ResponseWriter, r *http.Request) {
	db := getDBConnection()
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

	// Check if a file was uploaded
	file, handler, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		// Retrieve the filename associated with the item from the database
		var filename string
		err = db.QueryRow("SELECT filename FROM items WHERE id = $1", id).Scan(&filename)
		if err != nil {
			if err == sql.ErrNoRows {
				respondWithError(w, http.StatusNotFound, "Item not found")
			} else {
				respondWithError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		// Delete the associated file from the server
		err = deleteFile(filename)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Upload the file and get the generated filename
		filename, err := uploadFile(file, handler)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Update the filename in the database
		_, err = db.Exec("UPDATE items SET filename=$1 WHERE id=$2", filename, id)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	_, err = db.Exec("UPDATE items SET name=$1, price=$2 WHERE id=$3", name, price, id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "Update item successfully"})
}

func deleteItem(w http.ResponseWriter, r *http.Request) {
	db := getDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	// Retrieve the filename associated with the item from the database
	var filename string
	err = db.QueryRow("SELECT filename FROM items WHERE id = $1", id).Scan(&filename)
	if err != nil {
		if err == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Delete the associated file from the server
	err = deleteFile(filename)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Delete the item from the database
	_, err = db.Exec("DELETE FROM items WHERE id=$1", id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
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

func getDBConnection() *sql.DB {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	return db
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
