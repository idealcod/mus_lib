package main

import (
	"encoding/json"
	"net/http"
)

func main() {
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		group := r.URL.Query().Get("group")
		song := r.URL.Query().Get("song")
		if group == "" || song == "" {
			http.Error(w, "Missing group or song", http.StatusBadRequest)
			return
		}

		response := map[string]string{
			"releaseDate": "16.07.2006",
			"text":        "Ooh baby, don't you know I suffer?\n\nOoh baby, can you hear me moan?",
			"link":        "https://www.youtube.com/watch?v=Xsp3_a-PMTw",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})

	http.ListenAndServe(":8081", nil)
}
