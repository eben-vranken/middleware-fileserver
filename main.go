package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/{file}", getFile)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe("128.0.0.1:8080", nil))
}

func getFile(w http.ResponseWriter, req *http.Request) {
	LoggingMiddleware()
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, duration)
	})
}
