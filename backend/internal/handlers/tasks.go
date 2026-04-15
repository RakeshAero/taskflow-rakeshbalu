package handlers

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/repository"
)

// TaskHandler handles all /tasks and /projects/:id/tasks routes.
type TaskHandler struct {
	tasks    *repository.TaskRepository
	projects *repository.ProjectRepository
}

func NewTaskHandler(tasks *repository.TaskRepository, projects *repository.ProjectRepository) *TaskHandler {
	return &TaskHandler{tasks: tasks, projects: projects}
}

// List handles GET /projects/:id/tasks
// Supports optional query params: ?status=todo  ?assignee=<uuid>
func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "id")

	// Verify the project actually exists before listing its tasks.
	// Without this check, a request for a non-existent project returns []
	// instead of 404 — confusing for API consumers.
	if _, err := h.projects.GetByID(r.Context(), projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	// Read optional filter params from the URL query string.
	// chi.URLParam is for path params (:id); r.URL.Query().Get() is for query params (?key=val).
	filters := models.TaskFilters{
		Status:     models.TaskStatus(r.URL.Query().Get("status")),
		AssigneeID: r.URL.Query().Get("assignee"),
	}

	tasks, err := h.tasks.ListByProject(r.Context(), projectID, filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not fetch tasks")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tasks": tasks,
	})
}

// Create handles POST /projects/:id/tasks
// Adds a new task to the given project. Status defaults to "todo".
func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	projectID := chi.URLParam(r, "id")

	// Verify project exists before inserting a task into it.
	if _, err := h.projects.GetByID(r.Context(), projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	var input models.CreateTaskInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	task, err := h.tasks.Create(r.Context(), projectID, userID, &input)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// Update handles PATCH /tasks/:id
// Any authenticated user can update a task's fields.
// Authorization rule from the spec: no ownership restriction on update —
// any project member can change status/priority/assignee.
func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	// Confirm task exists before attempting update.
	if _, err := h.tasks.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch task")
		return
	}

	var input models.UpdateTaskInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	updated, err := h.tasks.Update(r.Context(), id, &input)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /tasks/:id
// Only the project owner OR the task creator can delete.
func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	id := chi.URLParam(r, "id")

	// Fetch task to get project_id and created_by for the auth check.
	task, err := h.tasks.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch task")
		return
	}

	// Fetch the parent project to get its owner.
	project, err := h.projects.GetByID(r.Context(), task.ProjectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not fetch project")
		return
	}

	// Authorization: must be project owner OR task creator.
	isProjectOwner := project.OwnerID == userID
	isTaskCreator := task.CreatedBy == userID

	if !isProjectOwner && !isTaskCreator {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.tasks.Delete(r.Context(), id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not delete task")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
