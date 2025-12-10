package main

import (
	"fmt"
	"log"
	"math"

	"github.com/bartolsthoorn/gohighs/highs"
)

func main() {
	// Minimize: x + y
	// Subject to: x + y >= 1, 0 <= x,y <= 10
	model := highs.Model{
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 0.0},
		ColUpper: []float64{10.0, 10.0},
	}
	model.AddDenseRow(1.0, []float64{1.0, 1.0}, math.Inf(1)) // x + y >= 1

	solution, err := model.Solve(highs.WithOutput(false))
	if err != nil {
		log.Fatal(err)
	}

	if solution.IsOptimal() {
		fmt.Printf("x = %.2f, y = %.2f\n", solution.ColValues[0], solution.ColValues[1])
		fmt.Printf("Objective = %.2f\n", solution.Objective)
	}
}

