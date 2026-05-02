package response

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, statusCode int, err string, message string) {
	JSON(w, statusCode, ErrorResponse{
		Error:   err,
		Message: message,
	})
}
