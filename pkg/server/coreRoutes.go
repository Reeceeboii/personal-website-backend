package server

import "net/http"

// Redirect any landing page requests
func Root(writer http.ResponseWriter, r *http.Request) {
	http.Redirect(writer, r, "https://reecemercer.dev", http.StatusOK)
}

// 404 route
func FourOhFour(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "Not found", http.StatusNotFound)
}
