package models

import "time"

// Project represents a row in the projects table.
// Description is a pointer (*string) because it is nullable in the DB.
// A pointer can be nil (NULL) or point to a value — plain string can't represent NULL.
type Project struct {
	ID          string    `db:"id"          json:"id"`
	Name        string    `db:"name"        json:"name"`
	Description *string   `db:"description" json:"description"` // nullable
	OwnerID     string    `db:"owner_id"    json:"owner_id"`
	CreatedAt   time.Time `db:"created_at"  json:"created_at"`
}

// ProjectWithTasks is returned by GET /projects/:id.
// It embeds Project and adds a Tasks slice so a single endpoint returns
// everything the frontend needs — no extra round-trips.
type ProjectWithTasks struct {
	Project
	Tasks []Task `json:"tasks"`
}

// CreateProjectInput is what the client sends to POST /projects.
type CreateProjectInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"` // optional
}

func (i *CreateProjectInput) Validate() map[string]string {
	errs := map[string]string{}

	if i.Name == "" {
		errs["name"] = "is required"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// UpdateProjectInput is what the client sends to PATCH /projects/:id.
// Both fields are pointers so we can tell the difference between:
//   - field not sent at all  → nil  → don't update that column
//   - field sent as ""       → &""  → update to empty string (validation will catch this)
//
// This is the standard Go pattern for partial updates (PATCH semantics).
type UpdateProjectInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (i *UpdateProjectInput) Validate() map[string]string {
	errs := map[string]string{}

	// Name is optional to send, but if it IS sent it cannot be empty.
	if i.Name != nil && *i.Name == "" {
		errs["name"] = "cannot be empty"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}