package response

import (
	"encoding/json"
	"net/http"
)

func WriteJSON(writer http.ResponseWriter, data any, code int) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	if err := json.NewEncoder(writer).Encode(data); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}
}
