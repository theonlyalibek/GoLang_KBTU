package delivery

import (
	"fmt"
	"sort"
)

// Service owns in-memory tasks. It is synchronous and is not safe for concurrent use.
type Service struct {
	calculator Calculator
	tasks      map[int]Task
	nextID     int
}

// NewService requires a usable calculator implementation, not a typed nil pointer.
func NewService(calculator Calculator) (*Service, error) {
	if calculator == nil {
		return nil, ErrInvalidCalculator
	}
	return &Service{calculator: calculator, tasks: make(map[int]Task), nextID: 1}, nil
}

// Create validates before assigning an ID or changing the store.
func (s *Service) Create(input DeliveryInput) (Task, error) {
	if err := input.Validate(); err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	task := Task{ID: s.nextID, Input: input, Status: StatusPending}
	s.tasks[task.ID] = task
	s.nextID++
	return task, nil
}

// Get returns a value snapshot, so callers cannot mutate the stored task.
func (s *Service) Get(id int) (Task, error) {
	task, ok := s.tasks[id]
	if !ok {
		return Task{}, fmt.Errorf("get task %d: %w", id, ErrNotFound)
	}
	return task, nil
}

// List returns independent task snapshots ordered by ID.
func (s *Service) List() []Task {
	tasks := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})
	return tasks
}

// Process runs a pending task synchronously and persists its final state.
func (s *Service) Process(id int) (Task, error) {
	task, err := s.Get(id)
	if err != nil {
		return Task{}, fmt.Errorf("process task: %w", err)
	}
	if task.Status != StatusPending {
		return task, fmt.Errorf("process task %d: %w: got %s, want pending", id, ErrInvalidState, task.Status)
	}

	task.Status = StatusProcessing
	s.tasks[id] = task
	result, err := s.calculator.Calculate(task.Input)
	if err != nil {
		task.Status = StatusFailed
		s.tasks[id] = task
		return task, fmt.Errorf("process task %d: %w", id, err)
	}
	task.Result = result
	task.Status = StatusCompleted
	s.tasks[id] = task
	return task, nil
}
