package main

import (
	"log"
	"net/http"
	"time"
)

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		defer func() {
			duration := time.Since(start)
			log.Printf("%s %s - %s\n", r.Method, r.URL.Path, duration)
		}()

		next.ServeHTTP(w, r)
	})
}
