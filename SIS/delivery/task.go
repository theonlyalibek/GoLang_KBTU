// Package delivery calculates educational delivery quotes and manages tasks in memory.
package delivery

import (
	"errors"
	"fmt"
)

type Zone string

const (
	ZoneCity      Zone = "city"
	ZoneIntercity Zone = "intercity"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
)

var (
	ErrInvalidInput      = errors.New("invalid delivery input")
	ErrNotFound          = errors.New("task not found")
	ErrInvalidState      = errors.New("invalid task state")
	ErrInvalidCalculator = errors.New("calculator is required")
)

// DeliveryInput uses grams to avoid rounding errors from floating-point weights.
type DeliveryInput struct {
	WeightGrams int64
	Zone        Zone
}

func (input DeliveryInput) Validate() error {
	if input.WeightGrams <= 0 {
		return fmt.Errorf("%w: weight must be positive, got %d grams", ErrInvalidInput, input.WeightGrams)
	}
	if input.Zone != ZoneCity && input.Zone != ZoneIntercity {
		return fmt.Errorf("%w: unknown zone %q (use city or intercity)", ErrInvalidInput, input.Zone)
	}
	return nil
}

// DeliveryResult stores whole kilograms and whole Kazakhstani tenge.
type DeliveryResult struct {
	BillableKilograms int64
	TotalKZT          int64
}

// Task is a snapshot. Result is available only when Status is completed.
type Task struct {
	ID     int
	Input  DeliveryInput
	Status Status
	Result DeliveryResult
}
