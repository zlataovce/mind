package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/unrolled/render"
	"go.uber.org/zap"
	"net/http"
)

const UuidRegex = "(?i)^[0-9A-F]{8}-[0-9A-F]{4}-4[0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}$"

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	store, err := NewStore("mind.sqlite")
	if err != nil {
		logger.Fatal("could not load database", zap.String("database_path", "mind.sqlite"), zap.Error(err))
	}

	err = store.Initialize()
	if err != nil {
		logger.Fatal("could not initialize database", zap.String("database_path", "mind.sqlite"), zap.Error(err))
	}

	re := render.New()
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})
	r.Get(fmt.Sprintf("/api/{id:%s}", UuidRegex), func(w http.ResponseWriter, r *http.Request) {
		parsedId, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			re.JSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		} else {
			unlockKey := r.Header.Get("Unlock-Key")

			data, err := store.SelectData(parsedId, unlockKey)
			if err != nil {
				re.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			} else if data == nil {
				re.JSON(w, http.StatusNotFound, map[string]string{"error": "Entity not found"})
			} else {
				re.JSON(w, http.StatusOK, *data)
			}
		}
	})
	r.Post(fmt.Sprintf("/api/{id:%s}", UuidRegex), func(w http.ResponseWriter, r *http.Request) {
		if r.Body == nil {
			re.JSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Malformed request body"})
			return
		}

		parsedId, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			re.JSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		} else {
			body := make(map[string]string)

			err = json.NewDecoder(r.Body).Decode(&body)
			if err != nil {
				re.JSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Malformed request body"})
				return
			}

			data, ok := body["data"]
			if !ok {
				re.JSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "Malformed request body"})
				return
			}

			err := store.InsertOrReplaceData(Data{Id: parsedId, UnlockKey: r.Header.Get("Unlock-Key"), Data: data})
			if err != nil {
				re.JSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
			}
		}
	})

	if err := http.ListenAndServe(":2003", r); err != nil {
		logger.Fatal("http server errored", zap.Error(err))
	}
}
