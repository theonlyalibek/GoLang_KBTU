package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"deliverycalc/delivery"
)

func newMenuService(t *testing.T) *delivery.Service {
	t.Helper()
	service, err := delivery.NewService(delivery.TariffCalculator{})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestMenuMultipleTasks(t *testing.T) {
	service := newMenuService(t)
	input := "2\n1\n1500\ncity\n1\n3000\nintercity\n2\n3\n2\n4\n2\n4\n1\n2\n0\n"
	var out, errOut bytes.Buffer
	if code := runMenu(service, strings.NewReader(input), &out, &errOut); code != 0 {
		t.Fatalf("exit=%d; stderr=%q", code, errOut.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected error: %s", errOut.String())
	}
	for _, want := range []string{
		"Заданий пока нет.",
		"Задание #1 создано: pending",
		"Задание #2 создано: pending",
		"1 | 1500 | city | pending | -",
		"2 | 3000 | intercity | pending | -",
		"2 | 3000 | intercity | completed | 2500",
		"Задание #2: completed",
		"Расчётный вес: 3 кг",
		"Стоимость: 2500 KZT",
		"Задание #1: pending",
		"Результат пока отсутствует.",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %s", want, out.String())
		}
	}
	if strings.Index(out.String(), "1 | 1500") > strings.Index(out.String(), "2 | 3000") {
		t.Fatal("list is not sorted by ID")
	}
	tasks := service.List()
	if len(tasks) != 2 || tasks[0].Status != delivery.StatusPending || tasks[1].Status != delivery.StatusCompleted || tasks[1].Result.TotalKZT != 2500 {
		t.Fatalf("unexpected session state: %+v", tasks)
	}
}

func TestMenuRecoversFromInvalidInput(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
	}{
		{"unknown command", "9\n"},
		{"empty command", "\n"},
		{"invalid weight", "1\nheavy\n"},
		{"extra weight text", "1\n1500 grams\n"},
		{"weight overflow", "1\n9223372036854775808\n"},
		{"zero weight", "1\n0\ncity\n"},
		{"negative weight", "1\n-10\ncity\n"},
		{"invalid zone", "1\n1500\nabroad\n"},
		{"missing task to process", "3\n99\n"},
		{"missing task to retrieve", "4\n99\n"},
		{"invalid ID", "3\nabc\n"},
		{"negative ID", "4\n-1\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			service := newMenuService(t)
			var out, errOut bytes.Buffer
			input := tt.input + "1\n1500\ncity\n0\n"
			if code := runMenu(service, strings.NewReader(input), &out, &errOut); code != 0 {
				t.Fatalf("recoverable error returned exit %d", code)
			}
			if errOut.Len() == 0 {
				t.Fatal("invalid input did not produce an error")
			}
			tasks := service.List()
			if len(tasks) != 1 || tasks[0].ID != 1 || tasks[0].Input.WeightGrams != 1500 || tasks[0].Status != delivery.StatusPending {
				t.Fatalf("did not recover and create the next task: %+v; output=%s", tasks, out.String())
			}
		})
	}
}

func TestMenuRejectsRepeatedProcessingAndContinues(t *testing.T) {
	service := newMenuService(t)
	input := "1\n1500\ncity\n3\n1\n3\n1\n1\n3000\nintercity\n0\n"
	var out, errOut bytes.Buffer
	if code := runMenu(service, strings.NewReader(input), &out, &errOut); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	if !strings.Contains(errOut.String(), "invalid task state") {
		t.Fatalf("missing state error: %q", errOut.String())
	}
	tasks := service.List()
	if len(tasks) != 2 || tasks[0].Status != delivery.StatusCompleted || tasks[0].Result.TotalKZT != 1100 || tasks[1].Status != delivery.StatusPending {
		t.Fatalf("unexpected state after retry: %+v", tasks)
	}
}

func TestMenuEOFAtPrompts(t *testing.T) {
	for _, input := range []string{"", "1", "1\n", "1\n1500", "3", "4\n"} {
		t.Run(input, func(t *testing.T) {
			service := newMenuService(t)
			var out, errOut bytes.Buffer
			if code := runMenu(service, strings.NewReader(input), &out, &errOut); code != 0 || errOut.Len() != 0 {
				t.Fatalf("EOF exit=%d; stderr=%q", code, errOut.String())
			}
			if tasks := service.List(); len(tasks) != 0 {
				t.Fatalf("incomplete input created tasks: %+v", tasks)
			}
		})
	}
}

func TestMenuFinalLineWithoutNewline(t *testing.T) {
	service := newMenuService(t)
	var out, errOut bytes.Buffer
	input := " 1 \n 1500 \n city "
	if code := runMenu(service, strings.NewReader(input), &out, &errOut); code != 0 || errOut.Len() != 0 {
		t.Fatalf("exit=%d; stderr=%q", code, errOut.String())
	}
	tasks := service.List()
	if len(tasks) != 1 || tasks[0].Input != (delivery.DeliveryInput{WeightGrams: 1500, Zone: delivery.ZoneCity}) {
		t.Fatalf("last line was not accepted: %+v", tasks)
	}
}

func TestMenuReadFailure(t *testing.T) {
	for _, prefix := range []string{"", "0", "1\n", "1\n1500\n", "1\n1500\ncity", "3\n", "4\n"} {
		t.Run(prefix, func(t *testing.T) {
			service := newMenuService(t)
			input := io.MultiReader(strings.NewReader(prefix), iotest.ErrReader(io.ErrUnexpectedEOF))
			var out, errOut bytes.Buffer
			if code := runMenu(service, input, &out, &errOut); code == 0 {
				t.Fatal("read failure returned success")
			}
			if !strings.Contains(errOut.String(), io.ErrUnexpectedEOF.Error()) {
				t.Fatalf("missing read error: %q", errOut.String())
			}
			if tasks := service.List(); len(tasks) != 0 {
				t.Fatalf("read failure created a task: %+v", tasks)
			}
		})
	}
}
