package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	log.Println("Starting mock API on :8081")
	http.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s", r.URL.String())
		group := r.URL.Query().Get("group")
		song := r.URL.Query().Get("song")
		if group == "" || song == "" {
			log.Println("Missing group or song")
			http.Error(w, "Missing group or song", http.StatusBadRequest)
			return
		}

		response := map[string]string{
			"releaseDate": "16.07.2006",
			"text":        "Ooh baby, don't you know I suffer?\n\nOoh baby, can you hear me moan?",
			"link":        "https://www.youtube.com/watch?v=Xsp3_a-PMTw",
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Printf("Response sent for group=%s, song=%s", group, song)
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}
