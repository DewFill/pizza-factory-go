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

func WritePlainText(writer http.ResponseWriter, data string, code int) {
	writer.WriteHeader(code)
	writer.Header().Set("Content-Type", "plain/text")
	writer.Write([]byte(data))

}

func WriteOK(writer http.ResponseWriter) {
	writer.WriteHeader(200)
	writer.Write([]byte("OK"))
}

func WriteServerError(writer http.ResponseWriter) {
	http.Error(writer, "Server error", http.StatusInternalServerError)
}
