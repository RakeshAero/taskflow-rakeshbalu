package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// TaskRepository handles all DB queries for the tasks table.
type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// ListByProject returns all tasks for a project, with optional filters.
// Supports ?status=todo and ?assignee=<uuid> query params.
//
// We build the WHERE clause dynamically because the number of filter
// conditions varies. This is safe — we use parameterised placeholders
// ($1, $2) so there is no SQL injection risk.
func (r *TaskRepository) ListByProject(ctx context.Context, projectID string, filters models.TaskFilters) ([]models.Task, error) {
	// Start with conditions we always apply.
	conditions := []string{"project_id = $1"}
	args := []any{projectID}
	argIdx := 2

	if filters.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(filters.Status))
		argIdx++
	}

	if filters.AssigneeID != "" {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, filters.AssigneeID)
		argIdx++
	}

	query := fmt.Sprintf(
		"SELECT * FROM tasks WHERE %s ORDER BY created_at DESC",
		joinClauses(conditions),
	)

	var tasks []models.Task
	if err := r.db.SelectContext(ctx, &tasks, query, args...); err != nil {
		return nil, fmt.Errorf("ListByProject: %w", err)
	}

	if tasks == nil {
		tasks = []models.Task{}
	}

	return tasks, nil
}

// Create inserts a new task into a project.
// Status always defaults to "todo" — clients cannot set it on creation.
// The DB column default also enforces this, but we're explicit here too.
func (r *TaskRepository) Create(ctx context.Context, projectID, createdBy string, input *models.CreateTaskInput) (*models.Task, error) {
	query := `
		INSERT INTO tasks (title, description, status, priority, project_id, created_by, assignee_id, due_date)
		VALUES ($1, $2, 'todo', $3, $4, $5, $6, $7)
		RETURNING *`

	var task models.Task
	err := r.db.GetContext(ctx, &task, query,
		input.Title,
		input.Description, // *string — nil becomes NULL
		input.Priority,
		projectID,
		createdBy,
		input.AssigneeID, // *string — nil becomes NULL
		input.DueDate,    // *time.Time — nil becomes NULL
	)
	if err != nil {
		return nil, fmt.Errorf("Create task: %w", err)
	}

	return &task, nil
}

// GetByID fetches a single task by its UUID.
// Returns ErrNotFound if no row matches.
func (r *TaskRepository) GetByID(ctx context.Context, id string) (*models.Task, error) {
	query := `SELECT * FROM tasks WHERE id = $1 LIMIT 1`

	var task models.Task
	err := r.db.GetContext(ctx, &task, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetByID task: %w", err)
	}

	return &task, nil
}

// Update applies a partial update to a task row.
// Only non-nil fields are written. updated_at is handled by the DB trigger.
//
// Special case: AssigneeID can be explicitly set to nil (unassign a task).
// When the client sends { "assignee_id": null }, Go decodes it as a nil
// *string. We still want to write NULL to the DB in this case — so we
// use a separate boolean flag to distinguish "not sent" from "sent as null".
func (r *TaskRepository) Update(ctx context.Context, id string, input *models.UpdateTaskInput) (*models.Task, error) {
	setClauses := []string{}
	args := []any{}
	argIdx := 1

	if input.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *input.Title)
		argIdx++
	}

	if input.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *input.Description)
		argIdx++
	}

	if input.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*input.Status))
		argIdx++
	}

	if input.Priority != nil {
		setClauses = append(setClauses, fmt.Sprintf("priority = $%d", argIdx))
		args = append(args, string(*input.Priority))
		argIdx++
	}

	// AssigneeID is a *string in UpdateTaskInput.
	// json.Unmarshal sets it to:
	//   nil           → field was NOT in the JSON body → skip
	//   &""           → impossible (UUIDs are never empty strings)
	//   &"some-uuid"  → update assignee
	//
	// To unassign a task, the client sends { "assignee_id": null }.
	// json.Unmarshal decodes JSON null into a Go nil pointer —
	// but we can't distinguish "null sent" from "field omitted" with a plain *string.
	//
	// Solution: UpdateTaskInput uses a custom nullable wrapper for AssigneeID.
	// See models/task.go — NullableString carries an explicit "was it set?" flag.
	if input.AssigneeID.Set {
		setClauses = append(setClauses, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, input.AssigneeID.Value) // Value is *string: nil = NULL
		argIdx++
	}

	if input.DueDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("due_date = $%d", argIdx))
		args = append(args, *input.DueDate)
		argIdx++
	}

	// Nothing to update — return current state.
	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	query := fmt.Sprintf(
		"UPDATE tasks SET %s WHERE id = $%d RETURNING *",
		joinClauses(setClauses),
		argIdx,
	)
	args = append(args, id)

	var task models.Task
	if err := r.db.GetContext(ctx, &task, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("Update task: %w", err)
	}

	return &task, nil
}

// Delete removes a task by ID.
// Returns ErrNotFound if no row matched.
func (r *TaskRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("Delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Delete task rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
