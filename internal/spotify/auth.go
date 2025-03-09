package spotify

import (
	"net/http"
	"net/url"
)

type Message struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	http.HandleFunc("/api/token", tokenHandler)
}

func tokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	refreshToken := r.URL.Query().Get("refresh_token")
	if refreshToken == "" {
		http.Error(w, "Missing refresh token", http.StatusBadRequest)
		return
	}

	data := url.Values{}
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm("https://accounts.spotify.com/api/token", data)
	if err != nil {
		http.Error(w, "Error making POST request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Error from the other server", resp.StatusCode)
		return
	}
}
