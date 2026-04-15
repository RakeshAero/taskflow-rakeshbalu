package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
)

// ProjectRepository handles all DB queries for the projects table.
type ProjectRepository struct {
	db *sqlx.DB
}

func NewProjectRepository(db *sqlx.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// ListByUser returns all projects the user either:
//   a) owns, OR
//   b) has at least one task assigned to them in
//
// This matches the spec: GET /projects lists "projects the current user owns
// or has tasks in". A UNION deduplicates automatically.
func (r *ProjectRepository) ListByUser(ctx context.Context, userID string) ([]models.Project, error) {
	query := `
		SELECT DISTINCT p.*
		FROM projects p
		WHERE p.owner_id = $1

		UNION

		SELECT DISTINCT p.*
		FROM projects p
		JOIN tasks t ON t.project_id = p.id
		WHERE t.assignee_id = $1

		ORDER BY created_at DESC`

	// db.SelectContext scans multiple rows into a slice — no manual loop needed.
	// If no rows match it returns an empty slice (not an error).
	var projects []models.Project
	if err := r.db.SelectContext(ctx, &projects, query, userID); err != nil {
		return nil, fmt.Errorf("ListByUser: %w", err)
	}

	// Never return nil — return empty slice so the handler encodes [] not null.
	if projects == nil {
		projects = []models.Project{}
	}

	return projects, nil
}

// Create inserts a new project owned by ownerID and returns the full row.
func (r *ProjectRepository) Create(ctx context.Context, input *models.CreateProjectInput, ownerID string) (*models.Project, error) {
	query := `
		INSERT INTO projects (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING *`

	var project models.Project
	err := r.db.GetContext(ctx, &project, query, input.Name, input.Description, ownerID)
	if err != nil {
		return nil, fmt.Errorf("Create project: %w", err)
	}

	return &project, nil
}

// GetByID fetches a single project row by its UUID.
// Returns ErrNotFound if no row matches.
func (r *ProjectRepository) GetByID(ctx context.Context, id string) (*models.Project, error) {
	query := `SELECT * FROM projects WHERE id = $1 LIMIT 1`

	var project models.Project
	err := r.db.GetContext(ctx, &project, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetByID project: %w", err)
	}

	return &project, nil
}

// Update applies a partial update to a project row.
// Only non-nil fields in input are written — PATCH semantics.
//
// We build the SET clause dynamically because sqlx doesn't have a
// built-in partial-update helper. This is the standard Go pattern.
func (r *ProjectRepository) Update(ctx context.Context, id string, input *models.UpdateProjectInput) (*models.Project, error) {
	// args holds the values for $1, $2, ... in the final query.
	// We always end with the project id as the last arg (for the WHERE clause).
	setClauses := []string{}
	args := []any{}
	argIdx := 1 // Postgres placeholders are 1-indexed: $1, $2, ...

	if input.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *input.Name)
		argIdx++
	}

	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}

	// If the client sent an empty body (no fields), nothing changes.
	// We still return the current project so the response is consistent.
	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	// Build the final query by joining the SET clauses with commas.
	// Example result: "UPDATE projects SET name = $1 WHERE id = $2 RETURNING *"
	query := fmt.Sprintf(
		"UPDATE projects SET %s WHERE id = $%d RETURNING *",
		joinClauses(setClauses),
		argIdx,
	)
	args = append(args, id)

	var project models.Project
	if err := r.db.GetContext(ctx, &project, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("Update project: %w", err)
	}

	return &project, nil
}

// Delete removes a project and (via ON DELETE CASCADE) all its tasks.
// Returns ErrNotFound if the project doesn't exist.
func (r *ProjectRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM projects WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Delete project: %w", err)
	}

	// RowsAffected tells us whether the DELETE actually matched a row.
	// If 0, the project never existed — return 404.
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Delete project rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// GetTasksForProject returns all tasks belonging to a project.
// Called by the project handler's Get method to build the ProjectWithTasks response.
// Kept here rather than task_repo to avoid the handler needing both repos
// for a single GET /projects/:id call — though either design is valid.
func (r *ProjectRepository) GetTasksForProject(ctx context.Context, projectID string) ([]models.Task, error) {
	query := `SELECT * FROM tasks WHERE project_id = $1 ORDER BY created_at DESC`

	var tasks []models.Task
	if err := r.db.SelectContext(ctx, &tasks, query, projectID); err != nil {
		return nil, fmt.Errorf("GetTasksForProject: %w", err)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	return tasks, nil
}

// joinClauses joins a slice of SET clause strings with ", ".
// Kept here (not in a utils package) to avoid over-engineering.
func joinClauses(clauses []string) string {
	result := ""
	for i, c := range clauses {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}