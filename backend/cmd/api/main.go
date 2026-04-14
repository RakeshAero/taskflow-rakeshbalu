package main
import (
	"context"
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
 
	"github.com/RakeshAero/taskflow-rakeshbalu/internal/config"
	"github.com/RakeshAero/taskflow-rakeshbalu/internal/db"
	"github.com/RakeshAero/taskflow-rakeshbalu/internal/handlers"
	authmw "github.com/RakeshAero/taskflow-rakeshbalu/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/internal/repository"
)