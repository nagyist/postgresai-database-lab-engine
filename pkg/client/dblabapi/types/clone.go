/*
2020 © Postgres.ai
*/

// Package types provides request structures for Database Lab HTTP API.
package types

// CloneCreateRequest represents clone params of a create request.
type CloneCreateRequest struct {
	ID        string                     `json:"id"`
	Project   string                     `json:"project"`
	Protected bool                       `json:"protected"`
	DB        *DatabaseRequest           `json:"db"`
	Snapshot  *SnapshotCloneFieldRequest `json:"snapshot"`
}

// CloneUpdateRequest represents params of an update request.
type CloneUpdateRequest struct {
	Protected bool `json:"protected"`
}

// DatabaseRequest represents database params of a clone request.
type DatabaseRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// SnapshotCloneFieldRequest represents snapshot params of a create request.
type SnapshotCloneFieldRequest struct {
	ID string `json:"id"`
}
