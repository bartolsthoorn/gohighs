package highs

import (
	"math"
	"testing"
)

func almostEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

// TestLP tests a basic linear programming problem.
//
//	Min    f  =  x_0 +  x_1 + 3
//	s.t.                x_1 <= 7
//	       5 <=  x_0 + 2x_1 <= 15
//	       6 <= 3x_0 + 2x_1
//	0 <= x_0 <= 4; 1 <= x_1
func TestLP(t *testing.T) {
	model := Model{
		Offset:   3.0,
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 1.0},
		ColUpper: []float64{4.0, 1e30},
		ConstMatrix: []Nonzero{
			{0, 1, 1.0},
			{1, 0, 1.0},
			{1, 1, 2.0},
			{2, 0, 3.0},
			{2, 1, 2.0},
		},
		RowLower: []float64{-1e30, 5.0, 6.0},
		RowUpper: []float64{7.0, 15.0, 1e30},
	}

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	if !almostEqual(sol.ColValues[0], 0.5, 0.01) {
		t.Errorf("x0 = %f, expected 0.5", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 2.25, 0.01) {
		t.Errorf("x1 = %f, expected 2.25", sol.ColValues[1])
	}
	if !almostEqual(sol.Objective, 5.75, 0.01) {
		t.Errorf("Objective = %f, expected 5.75", sol.Objective)
	}
}

// TestLPMaximize tests a maximization LP problem.
func TestLPMaximize(t *testing.T) {
	model := Model{
		Maximize: true,
		Offset:   3.0,
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 1.0},
		ColUpper: []float64{4.0, 1e30},
		ConstMatrix: []Nonzero{
			{0, 1, 1.0},
			{1, 0, 1.0},
			{1, 1, 2.0},
			{2, 0, 3.0},
			{2, 1, 2.0},
		},
		RowLower: []float64{-1e30, 5.0, 6.0},
		RowUpper: []float64{7.0, 15.0, 1e30},
	}

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	if !almostEqual(sol.ColValues[0], 4.0, 0.01) {
		t.Errorf("x0 = %f, expected 4.0", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 5.5, 0.01) {
		t.Errorf("x1 = %f, expected 5.5", sol.ColValues[1])
	}
	if !almostEqual(sol.Objective, 12.5, 0.01) {
		t.Errorf("Objective = %f, expected 12.5", sol.Objective)
	}
}

// TestMIP tests a mixed-integer programming problem.
func TestMIP(t *testing.T) {
	model := Model{
		Maximize: true,
		Offset:   3.0,
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 1.0},
		ColUpper: []float64{4.0, 1e30},
		ConstMatrix: []Nonzero{
			{0, 1, 1.0},
			{1, 0, 1.0},
			{1, 1, 2.0},
			{2, 0, 3.0},
			{2, 1, 2.0},
		},
		RowLower: []float64{-1e30, 5.0, 6.0},
		RowUpper: []float64{7.0, 15.0, 1e30},
		VarTypes: []VariableType{Integer, Integer},
	}

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	if !almostEqual(sol.ColValues[0], 4.0, 0.01) {
		t.Errorf("x0 = %f, expected 4.0", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 5.0, 0.01) {
		t.Errorf("x1 = %f, expected 5.0", sol.ColValues[1])
	}
	if !almostEqual(sol.Objective, 12.0, 0.01) {
		t.Errorf("Objective = %f, expected 12.0", sol.Objective)
	}
}

// TestQP tests a quadratic programming problem.
//
//	minimize -x_2 - 3x_3 + (1/2)(2x_1^2 - 2x_1x_3 + 0.2x_2^2 + 2x_3^2)
//	subject to x_1 + x_3 <= 2
func TestQP(t *testing.T) {
	model := Model{
		ColCosts: []float64{0.0, -1.0, -3.0},
		ConstMatrix: []Nonzero{
			{0, 0, 1.0},
			{0, 2, 1.0},
		},
		RowLower: []float64{-1e30},
		RowUpper: []float64{2.0},
		Hessian: []Nonzero{
			{0, 0, 2.0},
			{0, 2, -1.0},
			{1, 1, 0.2},
			{2, 2, 2.0},
		},
	}

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	if !almostEqual(sol.ColValues[0], 0.5, 0.01) {
		t.Errorf("x0 = %f, expected 0.5", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 5.0, 0.01) {
		t.Errorf("x1 = %f, expected 5.0", sol.ColValues[1])
	}
	if !almostEqual(sol.ColValues[2], 1.5, 0.01) {
		t.Errorf("x2 = %f, expected 1.5", sol.ColValues[2])
	}
	if !almostEqual(sol.Objective, -5.25, 0.01) {
		t.Errorf("Objective = %f, expected -5.25", sol.Objective)
	}
}

// TestAddDenseRow tests the AddDenseRow convenience method.
func TestAddDenseRow(t *testing.T) {
	model := Model{
		Offset:   3.0,
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 1.0},
		ColUpper: []float64{4.0, 1.0e30},
	}
	model.AddDenseRow(-1.0e30, []float64{0.0, 1.0}, 7.0)
	model.AddDenseRow(5.0, []float64{1.0, 2.0}, 15.0)
	model.AddDenseRow(6.0, []float64{3.0, 2.0}, 1.0e30)

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	if !almostEqual(sol.ColValues[0], 0.5, 0.01) {
		t.Errorf("x0 = %f, expected 0.5", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 2.25, 0.01) {
		t.Errorf("x1 = %f, expected 2.25", sol.ColValues[1])
	}
}

// TestLowLevelAPI tests the low-level solver API.
func TestLowLevelAPI(t *testing.T) {
	solver, err := NewSolver()
	if err != nil {
		t.Fatalf("NewSolver failed: %v", err)
	}
	defer solver.Close()

	if err := solver.SetBoolOption("output_flag", false); err != nil {
		t.Fatalf("SetBoolOption failed: %v", err)
	}

	// Add variables: 0 <= x0 <= 10, 0 <= x1 <= 10
	if err := solver.AddVars([]float64{0.0, 0.0}, []float64{10.0, 10.0}); err != nil {
		t.Fatalf("AddVars failed: %v", err)
	}

	// Set objective: minimize x0 + x1
	if err := solver.SetColCosts([]float64{1.0, 1.0}); err != nil {
		t.Fatalf("SetColCosts failed: %v", err)
	}

	// Add constraint: 5 <= x0 + 2*x1 <= 15
	if err := solver.AddRow(5.0, 15.0, []int{0, 1}, []float64{1.0, 2.0}); err != nil {
		t.Fatalf("AddRow failed: %v", err)
	}

	sol, err := solver.Run()
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	// Minimum: x0 + 2*x1 = 5 (binding), minimize x0 + x1
	// Substituting x0 = 5 - 2*x1, minimize 5 - x1
	// Maximum x1 = 2.5 (from x0 >= 0), so x0 = 0, x1 = 2.5, objective = 2.5
	if !almostEqual(sol.ColValues[0], 0.0, 0.01) {
		t.Errorf("x0 = %f, expected 0.0", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 2.5, 0.01) {
		t.Errorf("x1 = %f, expected 2.5", sol.ColValues[1])
	}
	if !almostEqual(sol.Objective, 2.5, 0.01) {
		t.Errorf("Objective = %f, expected 2.5", sol.Objective)
	}
}

// TestDiceProblem tests the dice MIP example from the highs package.
// What is the maximum total face value of three dice A, B, C such that
// A - B = 2(B - C) where B > C?
func TestDiceProblem(t *testing.T) {
	model := Model{
		Maximize: true,
		VarTypes: []VariableType{Integer, Integer, Integer},
		ColCosts: []float64{1.0, 1.0, 1.0}, // Maximize A + B + C
		ColLower: []float64{1.0, 1.0, 1.0}, // Dice show at least 1
		ColUpper: []float64{6.0, 6.0, 6.0}, // Dice show at most 6
	}
	// A - 3B + 2C = 0 (from A - B = 2(B - C))
	model.AddDenseRow(0.0, []float64{1.0, -3.0, 2.0}, 0.0)
	// B - C >= 1 (B > C, so at least 1 difference)
	model.AddDenseRow(1.0, []float64{0.0, 1.0, -1.0}, math.Inf(1))

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal, got %s", sol.Status)
	}

	// Expected: A=6, B=4, C=3, total=13
	if !almostEqual(sol.ColValues[0], 6.0, 0.01) {
		t.Errorf("A = %f, expected 6.0", sol.ColValues[0])
	}
	if !almostEqual(sol.ColValues[1], 4.0, 0.01) {
		t.Errorf("B = %f, expected 4.0", sol.ColValues[1])
	}
	if !almostEqual(sol.ColValues[2], 3.0, 0.01) {
		t.Errorf("C = %f, expected 3.0", sol.ColValues[2])
	}
	if !almostEqual(sol.Objective, 13.0, 0.01) {
		t.Errorf("Objective = %f, expected 13.0", sol.Objective)
	}
}

// TestEmptyModel tests that an empty model returns optimal.
func TestEmptyModel(t *testing.T) {
	model := Model{}

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsOptimal() {
		t.Fatalf("Expected optimal for empty model, got %s", sol.Status)
	}
}

// TestInfeasible tests detection of infeasible models.
func TestInfeasible(t *testing.T) {
	model := Model{
		ColCosts: []float64{1.0},
		ColLower: []float64{0.0},
		ColUpper: []float64{10.0},
	}
	// x >= 5
	model.AddDenseRow(5.0, []float64{1.0}, math.Inf(1))
	// x <= 3
	model.AddDenseRow(math.Inf(-1), []float64{1.0}, 3.0)

	sol, err := model.Solve(WithOutput(false))
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if !sol.IsInfeasible() {
		t.Errorf("Expected infeasible, got %s", sol.Status)
	}
}

func TestSolverInfinity(t *testing.T) {
	solver, err := NewSolver()
	if err != nil {
		t.Fatalf("NewSolver failed: %v", err)
	}
	defer solver.Close()

	inf := solver.Infinity()
	if inf <= 0 || math.IsNaN(inf) {
		t.Errorf("Invalid infinity value: %f", inf)
	}
}

// Benchmarks

func BenchmarkLPSolve(b *testing.B) {
	model := Model{
		ColCosts: []float64{1.0, 1.0},
		ColLower: []float64{0.0, 0.0},
		ColUpper: []float64{10.0, 10.0},
	}
	model.AddDenseRow(1.0, []float64{1.0, 1.0}, 5.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := model.Solve(WithOutput(false))
		if err != nil {
			b.Fatal(err)
		}
	}
}
