package handlers

import (
	"encoding/json"
	"net/http"
)

// ContextKey is the unique key that represents a context value
type ContextKey string

func (c ContextKey) String() string {
	return "context key " + string(c)
}

// message maps message into JSON formatted string
func message(message string) map[string]interface{} {
	return map[string]interface{}{"message": message}
}

// errorMessage maps message and error code into JSON formatted string
func errorMessage(errorCode int, message string) map[string]interface{} {
	return map[string]interface{}{"errorCode": errorCode, "message": message}
}

// dataMessage maps data and message into JSON formatted string
func dataMessage(data interface{}, message string) map[string]interface{} {
	return map[string]interface{}{"message": message, "data": data}
}

// respond encodes a JSON response to a http request
func respond(w http.ResponseWriter, data map[string]interface{}) {
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// respondWithStatus encodes a JSON response to a http request and modifies response status code
func respondWithStatus(w http.ResponseWriter, statusCode int, data map[string]interface{}) {
	w.WriteHeader(statusCode)
	respond(w, data)
}
