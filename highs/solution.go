package highs

// Solution contains the results from solving an optimization model.
type Solution struct {
	// Status indicates the outcome of the solve.
	Status ModelStatus

	// ColValues contains the primal solution values for each column (variable).
	ColValues []float64

	// ColDuals contains the dual solution values for each column.
	// Only populated for LP problems.
	ColDuals []float64

	// RowValues contains the primal solution values for each row (constraint).
	RowValues []float64

	// RowDuals contains the dual solution values for each row.
	// Only populated for LP problems.
	RowDuals []float64

	// ColBasis contains the basis status for each column.
	// Only populated when a basis is available.
	ColBasis []BasisStatus

	// RowBasis contains the basis status for each row.
	// Only populated when a basis is available.
	RowBasis []BasisStatus

	// Objective is the value of the objective function at the solution.
	Objective float64
}

// IsOptimal returns true if the solution is optimal.
func (s *Solution) IsOptimal() bool {
	return s.Status == ModelStatusOptimal
}

// IsInfeasible returns true if the model is infeasible.
func (s *Solution) IsInfeasible() bool {
	return s.Status == ModelStatusInfeasible ||
		s.Status == ModelStatusUnboundedOrInfeasible
}

// IsUnbounded returns true if the model is unbounded.
func (s *Solution) IsUnbounded() bool {
	return s.Status == ModelStatusUnbounded ||
		s.Status == ModelStatusUnboundedOrInfeasible
}

// IsTimeLimit returns true if the solve terminated due to time limit.
func (s *Solution) IsTimeLimit() bool {
	return s.Status == ModelStatusTimeLimit
}

// HasSolution returns true if the solution contains valid values.
func (s *Solution) HasSolution() bool {
	return s.Status.HasSolution()
}

// Value returns the solution value for a variable by index.
// Returns 0 if the index is out of range.
func (s *Solution) Value(index int) float64 {
	if index < 0 || index >= len(s.ColValues) {
		return 0
	}
	return s.ColValues[index]
}
