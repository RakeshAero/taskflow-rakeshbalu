package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/config"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/db"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/handlers"
	authmw "github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/repository"
)

func main(){
	route := chi.NewRouter()
	route.Get("/ping", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]string{
			"message" : "pong",
		})
	})

	http.ListenAndServe(":3000", route)
}