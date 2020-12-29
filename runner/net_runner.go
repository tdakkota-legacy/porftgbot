package runner

import (
	"context"
)

type Result struct {
	Replies []string `json:"replies"`
}

type Query struct {
	Prompt string `json:"prompt"`
	Length int    `json:"length"`
}

// NetRunner represents a neural network runner.
type NetRunner interface {
	Query(ctx context.Context, q Query) (Result, error)
}
