package entities

import "github.com/google/uuid"

type Service struct {
	ID   uuid.UUID
	Name string
}
