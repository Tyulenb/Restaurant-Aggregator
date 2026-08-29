package infrahttp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
)

func ParseJSON(r *http.Request, payload any) error {
	if r.Body == nil {
		return fmt.Errorf("The body is empty")
	}
	return json.NewDecoder(r.Body).Decode(payload)
}

func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(payload)
	if err != nil {
		slog.Default().Error("writingJson", "error", err)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: message})
}

func EstablishmentID(w http.ResponseWriter, r *http.Request) (string, bool) {
	establishmentID, ok := r.Context().Value(EstablishmentIDContextKey).(string)
	if !ok || establishmentID == "" {
		WriteUnauthorized(w)
		return "", false
	}
	return establishmentID, true
}

func OptionalBoolQuery(r *http.Request, name string) (bool, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return false, nil
	}
	return strconv.ParseBool(value)
}

func OptionalBoolQueryValue(r *http.Request, name string) (*bool, error) {
	values, ok := r.URL.Query()[name]
	if !ok || len(values) == 0 {
		return nil, nil
	}
	value, err := strconv.ParseBool(values[0])
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func WriteBadRequest(w http.ResponseWriter) {
	WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "Request syntax or parameter format is invalid.")
}

func WriteValidationError(w http.ResponseWriter) {
	WriteError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Request validation failed.")
}

func WriteNotFound(w http.ResponseWriter) {
	WriteError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found.")
}

func WriteUnauthorized(w http.ResponseWriter) {
	WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid partner API key.")
}
