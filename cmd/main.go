package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"inventory/api/handlers"
	"inventory/config"
	"inventory/database"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

const (
	uploadDir = "./uploads"
)

func main() {
	router := mux.NewRouter()

	router.HandleFunc("/item", handlers.GetItems).Methods("GET")
	router.HandleFunc("/item/{id}", handlers.GetItem).Methods("GET")
	router.HandleFunc("/item", handlers.AddItem).Methods("POST")
	router.HandleFunc("/item/{id}", handlers.UpdateItem).Methods("PUT")
	router.HandleFunc("/item/{id}", handlers.DelItem).Methods("DELETE")

	// Buat handler untuk serve file dari folder "uploads"
	config.ServeUploads(router)

	// Buat folder upload jika belum ada
	if _, err := os.Stat(config.UploadDir); os.IsNotExist(err) {
		os.Mkdir(config.UploadDir, 0755)
	}

	db := database.GetDBConnection()
	defer db.Close()

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
