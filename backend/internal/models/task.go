package models

import (
	"encoding/json"
	"time"
)

// TaskStatus and TaskPriority are string-based enums.
// Defining them as named types (not plain strings) lets the compiler
// catch mistakes like passing a priority where a status is expected.
type TaskStatus string
type TaskPriority string

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"

	PriorityLow    TaskPriority = "low"
	PriorityMedium TaskPriority = "medium"
	PriorityHigh   TaskPriority = "high"
)

// validStatuses and validPriorities are used by Validate() below.
// Defined as maps for O(1) lookup — faster than looping a slice.
var validStatuses = map[TaskStatus]bool{
	StatusTodo:       true,
	StatusInProgress: true,
	StatusDone:       true,
}

var validPriorities = map[TaskPriority]bool{
	PriorityLow:    true,
	PriorityMedium: true,
	PriorityHigh:   true,
}

// Task represents a row in the tasks table.
// Several fields are nullable → pointer types:
//   - *string        for AssigneeID (task may be unassigned)
//   - *string        for Description (optional text)
//   - *time.Time     for DueDate (optional date)
//
// UpdatedAt uses *time.Time so sqlx handles NULL correctly on fresh rows
// before any update has been made (though in practice the DB default covers this).
type Task struct {
	ID          string       `db:"id"          json:"id"`
	Title       string       `db:"title"       json:"title"`
	Description *string      `db:"description" json:"description"` // nullable
	Status      TaskStatus   `db:"status"      json:"status"`
	Priority    TaskPriority `db:"priority"    json:"priority"`
	ProjectID   string       `db:"project_id"  json:"project_id"`
	AssigneeID  *string      `db:"assignee_id" json:"assignee_id"` // nullable
	DueDate     *time.Time   `db:"due_date"    json:"due_date"`    // nullable
	CreatedAt   time.Time    `db:"created_at"  json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"  json:"updated_at"`
}

// CreateTaskInput is what the client sends to POST /projects/:id/tasks.
type CreateTaskInput struct {
	Title       string       `json:"title"`
	Description *string      `json:"description"`  // optional
	Priority    TaskPriority `json:"priority"`     // required
	AssigneeID  *string      `json:"assignee_id"`  // optional
	DueDate     *time.Time   `json:"due_date"`     // optional
	// Status is NOT in the create input — new tasks always start as "todo".
	// The client cannot set status on creation, only via PATCH later.
}

func (i *CreateTaskInput) Validate() map[string]string {
	errs := map[string]string{}

	if i.Title == "" {
		errs["title"] = "is required"
	}
	if i.Priority == "" {
		errs["priority"] = "is required"
	} else if !validPriorities[i.Priority] {
		errs["priority"] = "must be one of: low, medium, high"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// NullableString solves the "null vs omitted" problem for PATCH endpoints.
//
// With a plain *string in a JSON struct:
//   { "assignee_id": "uuid" }  → *string points to "uuid"  ← update
//   (field not in body)        → *string is nil             ← skip (correct)
//   { "assignee_id": null }    → *string is nil             ← also skip (WRONG — we want to set NULL)
//
// NullableString adds a Set flag so we can tell the difference:
//   { "assignee_id": null }    → Set=true,  Value=nil   ← write NULL to DB
//   (field not in body)        → Set=false, Value=nil   ← skip column entirely
type NullableString struct {
	Value *string // nil means SQL NULL
	Set   bool    // true if the field was present in the JSON body
}

// UnmarshalJSON implements json.Unmarshaler.
// Called automatically when json.Decode encounters this field.
func (ns *NullableString) UnmarshalJSON(data []byte) error {
	ns.Set = true // field was present in the body
	if string(data) == "null" {
		ns.Value = nil // explicit null → unassign
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ns.Value = &s
	return nil
}

// UpdateTaskInput is what the client sends to PATCH /tasks/:id.
// Every field is a pointer — only fields that are non-nil get updated.
// This gives true PATCH behaviour: send only what you want to change.
//
// Example: to mark a task done, client sends just: { "status": "done" }
// AssigneeID, Title, etc. stay untouched because they are nil.
//
// AssigneeID uses NullableString (not *string) so the repo can distinguish
// between "field omitted" and "field explicitly set to null" (unassign).
type UpdateTaskInput struct {
	Title       *string        `json:"title"`
	Description *string        `json:"description"`
	Status      *TaskStatus    `json:"status"`
	Priority    *TaskPriority  `json:"priority"`
	AssigneeID  NullableString `json:"assignee_id"`
	DueDate     *time.Time     `json:"due_date"`
}

func (i *UpdateTaskInput) Validate() map[string]string {
	errs := map[string]string{}

	if i.Title != nil && *i.Title == "" {
		errs["title"] = "cannot be empty"
	}
	if i.Status != nil && !validStatuses[*i.Status] {
		errs["status"] = "must be one of: todo, in_progress, done"
	}
	if i.Priority != nil && !validPriorities[*i.Priority] {
		errs["priority"] = "must be one of: low, medium, high"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// TaskFilters is populated from URL query params: ?status=todo&assignee=uuid
// Used by the repository to build the WHERE clause dynamically.
type TaskFilters struct {
	Status     TaskStatus // empty string means "no filter"
	AssigneeID string     // empty string means "no filter"
}