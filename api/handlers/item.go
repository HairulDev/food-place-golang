package handlers

import (
	"inventory/api/models"
	"inventory/database"
	"inventory/pkg"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func GetItems(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	items, err := models.GetItems(db)
	if err != nil {
		pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pkg.RespondWithJSON(w, http.StatusOK, items)
}

func GetItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	item, err := models.GetItem(db, id)
	if err != nil {
		if err == models.ErrItemNotFound {
			pkg.RespondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	pkg.RespondWithJSON(w, http.StatusOK, item)
}

func AddItem(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Invalid price")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Failed to read file")
		return
	}
	defer file.Close()

	filename, err := pkg.UploadFile(file, handler)
	if err != nil {
		pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	db := database.GetDBConnection()
	defer db.Close()

	err = models.AddItem(db, name, price, filename)
	if err != nil {
		pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pkg.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Add item successfully"})
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	// Get the item data from the form values
	name := r.FormValue("name")
	priceStr := r.FormValue("price")
	price, err := strconv.Atoi(priceStr)
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Invalid price")
		return
	}

	err = models.UpdateItem(db, id, name, price, r, w, pkg.UploadFile, pkg.DeleteFile)
	if err != nil {
		if err == models.ErrItemNotFound {
			pkg.RespondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	pkg.RespondWithJSON(w, http.StatusOK, map[string]interface{}{"message": "Update item successfully"})
}

func DelItem(w http.ResponseWriter, r *http.Request) {
	db := database.GetDBConnection()
	defer db.Close()

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		pkg.RespondWithError(w, http.StatusBadRequest, "Invalid item ID")
		return
	}

	filename, err := models.DelItem(db, id)
	if err != nil {
		if err == models.ErrItemNotFound {
			pkg.RespondWithError(w, http.StatusNotFound, "Item not found")
		} else {
			pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	err = pkg.DeleteFile(filename)
	if err != nil {
		pkg.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pkg.RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Item deleted successfully"})
}
