package delivery

import (
	"errors"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	s, err := NewService(TariffCalculator{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	return s
}

func TestNewServiceRejectsNilCalculator(t *testing.T) {
	s, err := NewService(nil)
	if s != nil || !errors.Is(err, ErrInvalidCalculator) {
		t.Fatalf("NewService(nil) = %v, %v; want nil, ErrInvalidCalculator", s, err)
	}
}

func TestServiceCreateAndGet(t *testing.T) {
	s := newTestService(t)
	input := DeliveryInput{1500, ZoneCity}
	task, err := s.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	if task.ID <= 0 || task.Input != input || task.Status != StatusPending || task.Result != (DeliveryResult{}) {
		t.Fatalf("unexpected created task: %+v", task)
	}
	got, err := s.Get(task.ID)
	if err != nil || got != task {
		t.Fatalf("Get(%d) = %+v, %v; want %+v", task.ID, got, err, task)
	}
}

func TestServiceRejectsInvalidInput(t *testing.T) {
	s := newTestService(t)
	for _, input := range []DeliveryInput{{0, ZoneCity}, {-100, ZoneIntercity}, {1000, ""}, {1000, "unknown"}} {
		task, err := s.Create(input)
		if !errors.Is(err, ErrInvalidInput) || task != (Task{}) {
			t.Fatalf("Create(%+v) = %+v, %v; want zero task, ErrInvalidInput", input, task, err)
		}
	}
	if _, err := s.Get(1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid inputs must not be stored: %v", err)
	}
	task, err := s.Create(DeliveryInput{1000, ZoneCity})
	if err != nil || task.ID != 1 {
		t.Fatalf("invalid inputs consumed an ID: task=%+v, error=%v", task, err)
	}
}

func TestServiceMissingTask(t *testing.T) {
	s := newTestService(t)
	for _, id := range []int{-1, 0, 42} {
		got, err := s.Get(id)
		if !errors.Is(err, ErrNotFound) || got != (Task{}) {
			t.Errorf("Get(%d) = %+v, %v; want zero task, ErrNotFound", id, got, err)
		}
		got, err = s.Process(id)
		if !errors.Is(err, ErrNotFound) || got != (Task{}) {
			t.Errorf("Process(%d) = %+v, %v; want zero task, ErrNotFound", id, got, err)
		}
	}
}

func TestServiceProcessAndPersistResult(t *testing.T) {
	s := newTestService(t)
	task, err := s.Create(DeliveryInput{1500, ZoneCity})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Process(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != task.ID || got.Input != task.Input || got.Status != StatusCompleted || got.Result != (DeliveryResult{2, 1100}) {
		t.Fatalf("unexpected processed task: %+v", got)
	}
	saved, err := s.Get(task.ID)
	if err != nil || saved != got {
		t.Fatalf("result not persisted: %+v, %v; want %+v", saved, err, got)
	}
	if _, err := s.Process(task.ID); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("processing completed task: error=%v, want ErrInvalidState", err)
	}
	after, err := s.Get(task.ID)
	if err != nil || after != saved {
		t.Fatalf("rejected processing changed task: %+v, %v", after, err)
	}
}

func TestServiceIndependentTasks(t *testing.T) {
	s := newTestService(t)
	first, err := s.Create(DeliveryInput{1500, ZoneCity})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(DeliveryInput{3000, ZoneIntercity})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID {
		t.Fatal("tasks share an ID")
	}
	if _, err := s.Process(second.ID); err != nil {
		t.Fatal(err)
	}
	untouched, err := s.Get(first.ID)
	if err != nil || untouched != first {
		t.Fatalf("processing second task changed first: %+v, %v", untouched, err)
	}
	if _, err := s.Process(first.ID); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		original Task
		want     DeliveryResult
	}{
		{first, DeliveryResult{2, 1100}},
		{second, DeliveryResult{3, 2500}},
	} {
		got, err := s.Get(tt.original.ID)
		if err != nil || got.Input != tt.original.Input || got.Status != StatusCompleted || got.Result != tt.want {
			t.Errorf("Get(%d) = %+v, %v; want input %+v, completed, result %+v", tt.original.ID, got, err, tt.original.Input, tt.want)
		}
	}
}

func TestServiceReturnsIndependentSnapshots(t *testing.T) {
	s := newTestService(t)
	input := DeliveryInput{1500, ZoneCity}
	created, err := s.Create(input)
	if err != nil {
		t.Fatal(err)
	}
	id := created.ID
	created.Input.WeightGrams = 9000
	created.Status = StatusCompleted
	pending, err := s.Get(id)
	if err != nil || pending.Input != input || pending.Status != StatusPending {
		t.Fatalf("Create returned mutable storage: %+v, %v", pending, err)
	}
	processed, err := s.Process(id)
	if err != nil {
		t.Fatal(err)
	}
	processed.Result.TotalKZT = -1
	fetched, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	fetched.Input.Zone = ZoneIntercity
	fetched.Result.BillableKilograms = 100
	saved, err := s.Get(id)
	if err != nil || saved.Input != input || saved.Result != (DeliveryResult{2, 1100}) || saved.Status != StatusCompleted {
		t.Fatalf("caller changed storage via a snapshot: %+v, %v", saved, err)
	}
}

// calculatorFunc lets tests observe the lifecycle at the actual calculation boundary.
type calculatorFunc func(DeliveryInput) (DeliveryResult, error)

func (f calculatorFunc) Calculate(input DeliveryInput) (DeliveryResult, error) {
	return f(input)
}

func TestServiceProcessingStateDuringCalculation(t *testing.T) {
	var s *Service
	var id int
	calculator := calculatorFunc(func(input DeliveryInput) (DeliveryResult, error) {
		task, err := s.Get(id)
		if err != nil || task.Status != StatusProcessing || task.Result != (DeliveryResult{}) {
			t.Fatalf("state during calculation: %+v, %v; want processing with no result", task, err)
		}
		if _, err := s.Process(id); !errors.Is(err, ErrInvalidState) {
			t.Fatalf("reentrant Process() error=%v, want ErrInvalidState", err)
		}
		return (TariffCalculator{}).Calculate(input)
	})
	var err error
	s, err = NewService(calculator)
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.Create(DeliveryInput{1500, ZoneCity})
	if err != nil {
		t.Fatal(err)
	}
	id = task.ID
	completed, err := s.Process(id)
	if err != nil || completed.Status != StatusCompleted || completed.Result != (DeliveryResult{2, 1100}) {
		t.Fatalf("Process() = %+v, %v", completed, err)
	}
}

func TestServiceCalculatorFailure(t *testing.T) {
	wantErr := errors.New("pricing unavailable")
	calculator := calculatorFunc(func(DeliveryInput) (DeliveryResult, error) {
		return DeliveryResult{}, wantErr
	})
	s, err := NewService(calculator)
	if err != nil {
		t.Fatal(err)
	}
	task, err := s.Create(DeliveryInput{1500, ZoneCity})
	if err != nil {
		t.Fatal(err)
	}
	failed, err := s.Process(task.ID)
	if !errors.Is(err, wantErr) || err == wantErr {
		t.Fatalf("Process() error=%v; want wrapped calculator error", err)
	}
	if failed.ID != task.ID || failed.Input != task.Input || failed.Status != StatusFailed || failed.Result != (DeliveryResult{}) {
		t.Fatalf("unexpected failed task: %+v", failed)
	}
	saved, err := s.Get(task.ID)
	if err != nil || saved != failed {
		t.Fatalf("failure state not persisted: %+v, %v", saved, err)
	}
	if _, err := s.Process(task.ID); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("retrying failed task: error=%v, want ErrInvalidState", err)
	}
}

func TestServiceListEmpty(t *testing.T) {
	s := newTestService(t)
	if got := s.List(); len(got) != 0 {
		t.Fatalf("empty List() = %+v", got)
	}
}

func TestServiceListSortedSnapshots(t *testing.T) {
	s := newTestService(t)
	want := make([]Task, 0, 3)
	for _, input := range []DeliveryInput{{1500, ZoneCity}, {3000, ZoneIntercity}, {1000, ZoneCity}} {
		task, err := s.Create(input)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, task)
	}
	processed, err := s.Process(want[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	want[1] = processed
	for attempt := 0; attempt < 20; attempt++ {
		got := s.List()
		if len(got) != len(want) {
			t.Fatalf("List() length=%d, want %d", len(got), len(want))
		}
		for i, task := range got {
			if task != want[i] {
				t.Fatalf("List()[%d] = %+v, want %+v", i, task, want[i])
			}
		}
		// Neither changing an element nor reordering this slice may change storage.
		got[0].Input.WeightGrams = 9999
		got[1].Result.TotalKZT = -1
		got[2].Status = StatusFailed
		got[0], got[2] = got[2], got[0]
	}
	for _, original := range want {
		got, err := s.Get(original.ID)
		if err != nil || got != original {
			t.Fatalf("List exposed stored task %d: %+v, %v", original.ID, got, err)
		}
	}
}

func TestServiceListDoesNotAllocateIDs(t *testing.T) {
	s := newTestService(t)
	s.List()
	first, err := s.Create(DeliveryInput{1000, ZoneCity})
	if err != nil {
		t.Fatal(err)
	}
	s.List()
	second, err := s.Create(DeliveryInput{2000, ZoneIntercity})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != 1 || second.ID != 2 {
		t.Fatalf("listing consumed IDs: first=%d, second=%d", first.ID, second.ID)
	}
}
