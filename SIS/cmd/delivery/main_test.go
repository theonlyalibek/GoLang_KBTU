package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunQuote(t *testing.T) {
	for _, tt := range []struct {
		name  string
		args  []string
		price string
	}{
		{"defaults", nil, "1100 KZT"},
		{"city", []string{"-weight", "1500", "-zone", "city"}, "1100 KZT"},
		{"intercity", []string{"-weight", "1500", "-zone", "intercity"}, "2000 KZT"},
		{"interactive disabled", []string{"-interactive=false"}, "1100 KZT"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tt.args, strings.NewReader(""), &out, &errOut); code != 0 {
				t.Fatalf("exit=%d; stderr=%q", code, errOut.String())
			}
			if errOut.Len() != 0 {
				t.Fatalf("unexpected stderr: %q", errOut.String())
			}
			for _, want := range []string{"Task #1: pending", "Task #1: completed", "2 kg", tt.price} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("output %q does not contain %q", out.String(), want)
				}
			}
		})
	}
}

func TestRunInvalidArguments(t *testing.T) {
	for _, tt := range []struct {
		name string
		args []string
	}{
		{"zero", []string{"-weight", "0"}},
		{"negative", []string{"-weight", "-1"}},
		{"unknown zone", []string{"-zone", "abroad"}},
		{"not a number", []string{"-weight", "heavy"}},
		{"integer overflow", []string{"-weight", "9223372036854775808"}},
		{"unknown flag", []string{"-unknown"}},
		{"positional argument", []string{"extra"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := run(tt.args, strings.NewReader(""), &out, &errOut); code == 0 {
				t.Fatal("invalid arguments returned success")
			}
			if errOut.Len() == 0 {
				t.Fatal("missing error message")
			}
			if out.Len() != 0 {
				t.Fatalf("invalid input created a task: %q", out.String())
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-h"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("help exit=%d", code)
	}
	for _, want := range []string{"Usage:", "-weight", "-zone", "-interactive"} {
		if !strings.Contains(errOut.String(), want) {
			t.Errorf("help %q missing %q", errOut.String(), want)
		}
	}
}

func TestRunInteractive(t *testing.T) {
	var out, errOut bytes.Buffer
	input := strings.NewReader("1\n1500\ncity\n2\n0\n")
	if code := run([]string{"-interactive"}, input, &out, &errOut); code != 0 || errOut.Len() != 0 {
		t.Fatalf("exit=%d; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(out.String(), "1 | 1500 | city | pending | -") {
		t.Fatalf("interactive flag did not run menu: %q", out.String())
	}
	if strings.Contains(out.String(), "completed") {
		t.Fatal("interactive mode automatically processed a task")
	}
}

func TestRunInteractiveRejectsQuoteFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-interactive", "-weight", "1500"},
		{"-interactive", "-zone", "city"},
	} {
		var out, errOut bytes.Buffer
		if code := run(args, strings.NewReader("0\n"), &out, &errOut); code == 0 {
			t.Fatalf("conflicting flags %v returned success", args)
		}
		if !strings.Contains(errOut.String(), "cannot be combined") {
			t.Fatalf("missing conflict message: %q", errOut.String())
		}
		if out.Len() != 0 {
			t.Fatalf("menu started with conflicting flags: %q", out.String())
		}
	}
}
