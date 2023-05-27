package config

import (
	"net/http"

	"github.com/gorilla/mux"
)

const UploadDir = "uploads"

func ServeUploads(router *mux.Router) {
	uploadsHandler := http.FileServer(http.Dir(UploadDir))
	router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", uploadsHandler))
}
