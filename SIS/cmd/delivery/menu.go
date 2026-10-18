package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"deliverycalc/delivery"
)

func runMenu(service *delivery.Service, in io.Reader, out, errOut io.Writer) int {
	scanner := bufio.NewScanner(in)
	for {
		fmt.Fprintln(out, "\n1. Создать доставку\n2. Показать все задания\n3. Обработать задание по ID\n4. Посмотреть задание по ID\n0. Выход")
		choice, ok := readLine(scanner, out, "Выбор: ")
		if !ok {
			return inputExitCode(scanner, errOut)
		}

		switch choice {
		case "0":
			return 0
		case "1":
			weightText, ok := readLine(scanner, out, "Вес в граммах: ")
			if !ok {
				return inputExitCode(scanner, errOut)
			}
			weight, err := strconv.ParseInt(weightText, 10, 64)
			if err != nil {
				fmt.Fprintln(errOut, "Ошибка: введите вес целым числом граммов.")
				continue
			}
			zone, ok := readLine(scanner, out, "Зона (city/intercity): ")
			if !ok {
				return inputExitCode(scanner, errOut)
			}
			task, err := service.Create(delivery.DeliveryInput{WeightGrams: weight, Zone: delivery.Zone(zone)})
			if err != nil {
				fmt.Fprintln(errOut, "Ошибка:", err)
				continue
			}
			fmt.Fprintf(out, "Задание #%d создано: %s\n", task.ID, task.Status)
		case "2":
			printTasks(out, service.List())
		case "3", "4":
			idText, ok := readLine(scanner, out, "ID задания: ")
			if !ok {
				return inputExitCode(scanner, errOut)
			}
			id, err := strconv.Atoi(idText)
			if err != nil {
				fmt.Fprintln(errOut, "Ошибка: ID должен быть целым числом.")
				continue
			}
			var task delivery.Task
			if choice == "3" {
				task, err = service.Process(id)
			} else {
				task, err = service.Get(id)
			}
			if err != nil {
				fmt.Fprintln(errOut, "Ошибка:", err)
				continue
			}
			printTask(out, task)
		default:
			fmt.Fprintln(errOut, "Ошибка: выберите 0, 1, 2, 3 или 4.")
		}
	}
}

func readLine(scanner *bufio.Scanner, out io.Writer, prompt string) (string, bool) {
	fmt.Fprint(out, prompt)
	// Scan can return a final token even when the reader failed.
	if !scanner.Scan() || scanner.Err() != nil {
		return "", false
	}
	return strings.TrimSpace(scanner.Text()), true
}

// EOF ends a session normally. A real input failure is reported to the caller.
func inputExitCode(scanner *bufio.Scanner, errOut io.Writer) int {
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(errOut, "Ошибка чтения:", err)
		return 1
	}
	return 0
}

func printTask(out io.Writer, task delivery.Task) {
	fmt.Fprintf(out, "Задание #%d: %s\n", task.ID, task.Status)
	fmt.Fprintf(out, "Вес: %d г; зона: %s\n", task.Input.WeightGrams, task.Input.Zone)
	if task.Status == delivery.StatusCompleted {
		fmt.Fprintf(out, "Расчётный вес: %d кг\nСтоимость: %d KZT\n", task.Result.BillableKilograms, task.Result.TotalKZT)
	} else {
		fmt.Fprintln(out, "Результат пока отсутствует.")
	}
}

func printTasks(out io.Writer, tasks []delivery.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(out, "Заданий пока нет.")
		return
	}
	fmt.Fprintln(out, "ID | Вес (г) | Зона | Статус | Стоимость (KZT)")
	for _, task := range tasks {
		price := "-"
		if task.Status == delivery.StatusCompleted {
			price = strconv.FormatInt(task.Result.TotalKZT, 10)
		}
		fmt.Fprintf(out, "%d | %d | %s | %s | %s\n", task.ID, task.Input.WeightGrams, task.Input.Zone, task.Status, price)
	}
}
