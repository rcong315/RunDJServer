package main

import (
    "fmt"
    "log"
    "net/http"
)

type Message struct {
    Status  string `json:"status"`
    Message string `json:"message"`
}

func main() {
    http.HandleFunc("/", homeHandler)
	http.HandleFunc("/thanks", thanksHandler)

    port := ":8080"
    fmt.Printf("Server starting on port %s\n", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatal(err)
    }
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    if r.URL.Path != "/" {
        http.NotFound(w, r)
        return
    }

    fmt.Fprintf(w, "RunDJ Backend")
}

func thanksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body><a>getsongbpm.com</a></body></html>")
}
