package delivery

import (
	"errors"
	"math"
	"testing"
)

func TestDeliveryInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   DeliveryInput
		wantErr bool
	}{
		{"city", DeliveryInput{1500, ZoneCity}, false},
		{"intercity", DeliveryInput{1, ZoneIntercity}, false},
		{"zero weight", DeliveryInput{0, ZoneCity}, true},
		{"negative weight", DeliveryInput{-1, ZoneCity}, true},
		{"missing zone", DeliveryInput{1000, ""}, true},
		{"unknown zone", DeliveryInput{1000, "abroad"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("Validate() error = %v, want ErrInvalidInput", err)
				}
			} else if err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestTariffCalculatorCalculate(t *testing.T) {
	tests := []struct {
		name  string
		input DeliveryInput
		want  DeliveryResult
	}{
		{"one gram", DeliveryInput{1, ZoneCity}, DeliveryResult{1, 800}},
		{"below kilogram", DeliveryInput{999, ZoneCity}, DeliveryResult{1, 800}},
		{"exact kilogram", DeliveryInput{1000, ZoneCity}, DeliveryResult{1, 800}},
		{"over kilogram", DeliveryInput{1001, ZoneCity}, DeliveryResult{2, 1100}},
		{"city example", DeliveryInput{1500, ZoneCity}, DeliveryResult{2, 1100}},
		{"intercity example", DeliveryInput{1500, ZoneIntercity}, DeliveryResult{2, 2000}},
		{"intercity exact", DeliveryInput{3000, ZoneIntercity}, DeliveryResult{3, 2500}},
		{"largest weight", DeliveryInput{math.MaxInt64, ZoneIntercity}, DeliveryResult{9223372036854776, 4611686018427389000}},
	}
	var calculator Calculator = TariffCalculator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := calculator.Calculate(tt.input)
			if err != nil {
				t.Fatalf("Calculate() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("Calculate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestTariffCalculatorRejectsInvalidInput(t *testing.T) {
	for _, input := range []DeliveryInput{{0, ZoneCity}, {-1, ZoneCity}, {1000, "unknown"}} {
		got, err := (TariffCalculator{}).Calculate(input)
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Calculate(%+v) error = %v, want ErrInvalidInput", input, err)
		}
		if got != (DeliveryResult{}) {
			t.Errorf("invalid input produced result %+v", got)
		}
	}
}
