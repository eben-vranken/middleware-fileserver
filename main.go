package main

import (
	"log"
	"net/http"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

const USERNAME string = "user"
const PASSWORD string = "admin"

func main() {
	mux := http.NewServeMux()

	mux.Handle("GET /", loggingMiddleware(authMiddleware(http.FileServer(http.Dir("./public")))))

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println(req.URL.Path, "executing logging middleware")
		start := time.Now()

		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, req)
		duration := time.Since(start)
		log.Printf("[%s] %s %s %d", req.Method, req.RequestURI, duration, recorder.statusCode)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println("Authenticating user")
		username, password, ok := req.BasicAuth()

		if !ok || username != USERNAME || password != PASSWORD {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, req)
	})
}
