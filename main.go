package main

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var mu sync.Mutex

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.statusCode = code
	sr.ResponseWriter.WriteHeader(code)
}

var registeredIps map[string]int = make(map[string]int)

const REQUESTS_LIMIT_PER_MINUTE = 100
const USERNAME string = "user"
const PASSWORD string = "admin"

func main() {
	mux := http.NewServeMux()

	mux.Handle("/", loggingMiddleware(corsMiddleware(rateLimitingMiddleware(authMiddleware(http.FileServer(http.Dir("./public")))))))

	ticker := time.NewTicker(time.Minute)

	go func() {
		for range ticker.C {
			resetRequestCount()
		}
	}()

	log.Println("Listening on :8080...")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", mux))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println(req.URL.Path, "Initializing logging middleware")
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

func rateLimitingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()

		log.Println("Logging IP and checking request count in last minute")

		userIp := strings.Split(req.RemoteAddr, ":")[0]
		_, ok := registeredIps[userIp]

		if !ok {
			registeredIps[userIp] = 0
		}

		registeredIps[userIp]++
		count := registeredIps[userIp]

		mu.Unlock()

		if count > REQUESTS_LIMIT_PER_MINUTE {
			log.Printf("USER BLOCKED: %s %d requests in the last minute", userIp, count)
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		log.Printf("User %s has requested server %d time(s) in the last minute", userIp, count)

		next.ServeHTTP(w, req)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Println("Initializing CORS headers")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization")

		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, req)
	})
}

func resetRequestCount() {
	log.Println("Reseting request counts")
	mu.Lock()
	defer mu.Unlock()

	for ip := range registeredIps {
		delete(registeredIps, ip)
	}
}
