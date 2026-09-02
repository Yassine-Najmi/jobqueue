package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func handleCreateJob(store *InMemoryStore, registry map[string]JobHandler, jobChan chan<- Job) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var job Job

		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			http.Error(w, "the input is invalid", http.StatusBadRequest)
			return
		}

		if errs := validateJob(job, registry); len(errs) > 0 {
			var sb strings.Builder
			for _, err := range errs {
				sb.WriteString(err.Error())
				sb.WriteString("\n")
			}

			http.Error(w, sb.String(), http.StatusBadRequest)
			return
		}

		created, err := store.Create(job)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		jobChan <- created

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)
	}
}

func handleGetJob(store *InMemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))

		if err != nil {
			http.Error(w, "the id wasn't a valid integer", http.StatusBadRequest)
			return
		}

		job, err := store.Get(id)
		if errors.Is(err, ErrJobNotFound) {
			http.Error(w, fmt.Sprint(ErrJobNotFound), http.StatusNotFound)
			return
		}

		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(job)
	}
}
