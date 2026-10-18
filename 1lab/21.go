package main

import "fmt"

func changeFirst(values []int) {
    values[0] = 100
}

func main() {
    numbers := []int{10, 20, 30}

    fmt.Println("До функции:", numbers)

    changeFirst(numbers)

    fmt.Println("После функции:", numbers)
}