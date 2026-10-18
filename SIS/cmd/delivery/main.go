package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"deliverycalc/delivery"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, in io.Reader, out, errOut io.Writer) int {
	flags := flag.NewFlagSet("delivery", flag.ContinueOnError)
	flags.SetOutput(errOut)
	weight := flags.Int64("weight", 1500, "parcel weight in grams (positive integer)")
	zone := flags.String("zone", "city", "delivery zone: city or intercity")
	interactive := flags.Bool("interactive", false, "open the interactive task menu")
	flags.Usage = func() {
		fmt.Fprintln(errOut, "Usage: delivery -weight 1500 -zone city")
		fmt.Fprintln(errOut, "       delivery -interactive")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(errOut, "error: positional arguments are not supported; use -weight/-zone or -interactive")
		return 1
	}
	if *interactive && flags.NFlag() > 1 {
		fmt.Fprintln(errOut, "error: -interactive cannot be combined with -weight or -zone")
		return 1
	}

	service, err := delivery.NewService(delivery.TariffCalculator{})
	if err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return 1
	}
	if *interactive {
		return runMenu(service, in, out, errOut)
	}
	task, err := service.Create(delivery.DeliveryInput{WeightGrams: *weight, Zone: delivery.Zone(*zone)})
	if err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "Task #%d: %s\n", task.ID, task.Status)

	if _, err := service.Process(task.ID); err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return 1
	}
	saved, err := service.Get(task.ID)
	if err != nil {
		fmt.Fprintln(errOut, "error:", err)
		return 1
	}
	fmt.Fprintf(out, "Task #%d: %s\n", saved.ID, saved.Status)
	fmt.Fprintf(out, "Weight: %d g; zone: %s\n", saved.Input.WeightGrams, saved.Input.Zone)
	fmt.Fprintf(out, "Billable weight: %d kg\n", saved.Result.BillableKilograms)
	fmt.Fprintf(out, "Total: %d KZT\n", saved.Result.TotalKZT)
	return 0
}
