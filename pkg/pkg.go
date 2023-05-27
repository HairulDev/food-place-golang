package pkg

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	uploadDir = "./uploads"
)

func UploadFile(file multipart.File, handler *multipart.FileHeader) (string, error) {
	fmt.Println("uploadFile called")
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

func DeleteFile(filename string) error {
	fmt.Println("deleteFile called")
	filePath := filepath.Join(uploadDir, filename)
	err := os.Remove(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	RespondWithJSON(w, statusCode, map[string]string{"error": message})
}

func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(response)
}
