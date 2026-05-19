package main

import (
	"log"
	"net/http"
	"time"
)

func main() {

	mux := http.NewServeMux()

	mux.Handle("GET /", loggingMiddleware(http.FileServer(http.Dir("./public"))))

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println(req.URL.Path, "executing middleware")
		start := time.Now()
		next.ServeHTTP(w, req)
		duration := time.Since(start)
		log.Printf("[%s] %s %s", req.Method, req.RequestURI, duration)
	})
}
