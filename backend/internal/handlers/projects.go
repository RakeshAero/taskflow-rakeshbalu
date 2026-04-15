package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/repository"
)

// ProjectHandler handles all /projects routes.
type ProjectHandler struct {
	projects *repository.ProjectRepository
}

func NewProjectHandler(projects *repository.ProjectRepository) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

// List handles GET /projects
// Returns all projects the authenticated user owns or has tasks in.
func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	projects, err := h.projects.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch projects")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projects": projects,
	})
}

// Create handles POST /projects
// Creates a new project owned by the authenticated user.
func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var input models.CreateProjectInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	project, err := h.projects.Create(r.Context(), &input, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create project")
		return
	}

	// 201 Created with the full project object so the client has the generated ID.
	writeJSON(w, http.StatusCreated, project)
}

// Get handles GET /projects/:id
// Returns project details including all its tasks.
func (h *ProjectHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	project, err := h.projects.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	// Fetch the project's tasks from the task repo.
	// We do this in the handler (not the project repo) to keep each repo
	// focused on its own table. The handler orchestrates across repos.
	tasks, err := h.projects.GetTasksForProject(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch tasks")
		return
	}

	writeJSON(w, http.StatusOK, models.ProjectWithTasks{
		Project: *project,
		Tasks:   tasks,
	})
}

// Update handles PATCH /projects/:id
// Only the project owner can update. Returns 403 for other users.
func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	// Fetch first to check ownership before attempting the update.
	project, err := h.projects.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	// 403 Forbidden — authenticated but not authorised.
	// Important: never return 404 here — that would hide the project's existence
	// from people who can see it. The spec says 403, so use 403.
	if project.OwnerID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var input models.UpdateProjectInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	updated, err := h.projects.Update(r.Context(), id, &input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update project")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /projects/:id
// Only the project owner can delete. Cascades to all tasks via the DB.
func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	// Ownership check before delete — same pattern as Update.
	project, err := h.projects.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	if project.OwnerID != userID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.projects.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete project")
		return
	}

	// 204 No Content — successful delete, no body.
	w.WriteHeader(http.StatusNoContent)
}