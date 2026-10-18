package delivery

// Calculator separates delivery pricing from task storage and lifecycle.
// Implementations must be usable non-nil values, and must return a zero result on error.
type Calculator interface {
	Calculate(DeliveryInput) (DeliveryResult, error)
}

// TariffCalculator uses fixed educational tariffs, not a carrier's actual prices.
type TariffCalculator struct{}

func (TariffCalculator) Calculate(input DeliveryInput) (DeliveryResult, error) {
	if err := input.Validate(); err != nil {
		return DeliveryResult{}, err
	}

	var baseKZT, perKilogramKZT int64
	switch input.Zone {
	case ZoneCity:
		baseKZT, perKilogramKZT = 500, 300
	case ZoneIntercity:
		baseKZT, perKilogramKZT = 1000, 500
	}

	// Division plus remainder avoids overflow from adding 999 to the weight.
	kilograms := input.WeightGrams / 1000
	if input.WeightGrams%1000 != 0 {
		kilograms++
	}

	return DeliveryResult{
		BillableKilograms: kilograms,
		TotalKZT:          baseKZT + kilograms*perKilogramKZT,
	}, nil
}
