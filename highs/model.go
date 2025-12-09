package highs

import "math"

// Model represents a high-level optimization model.
// It provides a convenient way to define LP, MIP, and QP problems
// without dealing with the low-level solver API directly.
//
// The model solves problems of the form:
//
//	Minimize (or Maximize): ColCosts · x + Offset + 0.5 * x' * Hessian * x
//	Subject to:             RowLower ≤ A·x ≤ RowUpper
//	And:                    ColLower ≤ x ≤ ColUpper
//
// Where A is the constraint matrix specified by ConstMatrix.
type Model struct {
	// Maximize indicates whether to maximize (true) or minimize (false).
	Maximize bool

	// Offset is a constant added to the objective function.
	Offset float64

	// ColCosts are the objective function coefficients for each variable.
	ColCosts []float64

	// ColLower are the lower bounds for each variable.
	// If empty or shorter than the number of variables, defaults to -∞.
	ColLower []float64

	// ColUpper are the upper bounds for each variable.
	// If empty or shorter than the number of variables, defaults to +∞.
	ColUpper []float64

	// RowLower are the lower bounds for each constraint.
	// Use NegInf() for no lower bound.
	RowLower []float64

	// RowUpper are the upper bounds for each constraint.
	// Use Inf() for no upper bound.
	RowUpper []float64

	// ConstMatrix defines the constraint matrix as a list of non-zero entries.
	// Each entry specifies (row, column, value).
	ConstMatrix []Nonzero

	// Hessian defines the Hessian matrix for quadratic programming.
	// Must be upper triangular. Each entry specifies (row, column, value).
	// For a term like 0.5*x_i*Q_ij*x_j, set Hessian[{i,j}] = Q_ij.
	Hessian []Nonzero

	// VarTypes specifies the type of each variable (continuous, integer, etc.).
	// If empty, all variables are treated as continuous.
	VarTypes []VariableType
}

// AddDenseRow adds a constraint to the model using a dense coefficient vector.
// Zero coefficients are automatically filtered out.
//
// Example:
//
//	model.AddDenseRow(1.0, []float64{1.0, 2.0, 0.0, 3.0}, 10.0)
//	// Adds constraint: 1.0 <= x0 + 2*x1 + 3*x3 <= 10.0
func (m *Model) AddDenseRow(lower float64, coeffs []float64, upper float64) {
	row := len(m.RowLower)
	m.RowLower = append(m.RowLower, lower)
	m.RowUpper = append(m.RowUpper, upper)

	for col, val := range coeffs {
		if val != 0.0 {
			m.ConstMatrix = append(m.ConstMatrix, Nonzero{
				Row: row,
				Col: col,
				Val: val,
			})
		}
	}
}

// AddSparseRow adds a constraint using sparse coefficient representation.
//
// Example:
//
//	model.AddSparseRow(1.0, []int{0, 1, 3}, []float64{1.0, 2.0, 3.0}, 10.0)
//	// Adds constraint: 1.0 <= x0 + 2*x1 + 3*x3 <= 10.0
func (m *Model) AddSparseRow(lower float64, cols []int, vals []float64, upper float64) {
	row := len(m.RowLower)
	m.RowLower = append(m.RowLower, lower)
	m.RowUpper = append(m.RowUpper, upper)

	for i, col := range cols {
		if vals[i] != 0.0 {
			m.ConstMatrix = append(m.ConstMatrix, Nonzero{
				Row: row,
				Col: col,
				Val: vals[i],
			})
		}
	}
}

// AddEqRow adds an equality constraint: sum(coeffs * x) = rhs.
func (m *Model) AddEqRow(coeffs []float64, rhs float64) {
	m.AddDenseRow(rhs, coeffs, rhs)
}

// AddLeRow adds a less-than-or-equal constraint: sum(coeffs * x) <= rhs.
func (m *Model) AddLeRow(coeffs []float64, rhs float64) {
	m.AddDenseRow(math.Inf(-1), coeffs, rhs)
}

// AddGeRow adds a greater-than-or-equal constraint: sum(coeffs * x) >= rhs.
func (m *Model) AddGeRow(coeffs []float64, rhs float64) {
	m.AddDenseRow(rhs, coeffs, math.Inf(1))
}

// NumVars returns the number of variables in the model.
func (m *Model) NumVars() int {
	maxCol := -1
	for _, nz := range m.ConstMatrix {
		if nz.Col > maxCol {
			maxCol = nz.Col
		}
	}
	for _, nz := range m.Hessian {
		if nz.Col > maxCol {
			maxCol = nz.Col
		}
	}
	if len(m.ColCosts) > maxCol+1 {
		return len(m.ColCosts)
	}
	if len(m.ColLower) > maxCol+1 {
		return len(m.ColLower)
	}
	if len(m.ColUpper) > maxCol+1 {
		return len(m.ColUpper)
	}
	if len(m.VarTypes) > maxCol+1 {
		return len(m.VarTypes)
	}
	return maxCol + 1
}

// NumConstraints returns the number of constraints in the model.
func (m *Model) NumConstraints() int {
	maxRow := -1
	for _, nz := range m.ConstMatrix {
		if nz.Row > maxRow {
			maxRow = nz.Row
		}
	}
	if len(m.RowLower) > maxRow+1 {
		return len(m.RowLower)
	}
	if len(m.RowUpper) > maxRow+1 {
		return len(m.RowUpper)
	}
	return maxRow + 1
}

// Solve builds and solves the model, returning the solution.
//
// Options can be set using SolveOptions:
//
//	solution, err := model.Solve(
//		highs.WithTimeLimit(60),
//		highs.WithMIPGap(0.01),
//		highs.WithOutput(false),
//	)
func (m *Model) Solve(opts ...SolveOption) (*Solution, error) {
	solver, err := NewSolver()
	if err != nil {
		return nil, err
	}
	defer solver.Close()

	// Apply options
	cfg := defaultSolveConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	if err := cfg.apply(solver); err != nil {
		return nil, err
	}

	// Determine dimensions
	numCol := m.NumVars()
	numRow := m.NumConstraints()

	if numCol == 0 {
		return &Solution{Status: ModelStatusOptimal}, nil
	}

	// Prepare column data with defaults
	colCosts, err := expandSlice(numCol, m.ColCosts, 0.0)
	if err != nil {
		return nil, newErrorMsg("Solve", "inconsistent ColCosts length")
	}
	colLower, err := expandSlice(numCol, m.ColLower, math.Inf(-1))
	if err != nil {
		return nil, newErrorMsg("Solve", "inconsistent ColLower length")
	}
	colUpper, err := expandSlice(numCol, m.ColUpper, math.Inf(1))
	if err != nil {
		return nil, newErrorMsg("Solve", "inconsistent ColUpper length")
	}

	// Prepare row data with defaults
	rowLower, err := expandSlice(numRow, m.RowLower, math.Inf(-1))
	if err != nil {
		return nil, newErrorMsg("Solve", "inconsistent RowLower length")
	}
	rowUpper, err := expandSlice(numRow, m.RowUpper, math.Inf(1))
	if err != nil {
		return nil, newErrorMsg("Solve", "inconsistent RowUpper length")
	}

	// Convert constraint matrix to CSR format
	aStart, aIndex, aValue, err := nonzerosToCSR(m.ConstMatrix, false)
	if err != nil {
		return nil, err
	}

	// Prepare variable types
	varTypes := m.VarTypes
	if len(varTypes) > 0 && len(varTypes) != numCol {
		expanded := make([]VariableType, numCol)
		copy(expanded, varTypes)
		varTypes = expanded
	}

	// Pass the model
	err = solver.PassModel(
		numCol, numRow,
		colCosts, colLower, colUpper,
		rowLower, rowUpper,
		aStart, aIndex, aValue,
		varTypes,
		m.Maximize,
		m.Offset,
	)
	if err != nil {
		return nil, err
	}

	// Add Hessian for QP if present
	if len(m.Hessian) > 0 {
		hStart, hIndex, hValue, err := nonzerosToCSR(m.Hessian, true)
		if err != nil {
			return nil, err
		}
		if err := solver.PassHessian(numCol, hStart, hIndex, hValue); err != nil {
			return nil, err
		}
	}

	// Solve
	return solver.Run()
}

// SolveOption configures the solver behavior.
type SolveOption func(*solveConfig)

type solveConfig struct {
	output      *bool
	timeLimit   *float64
	mipAbsGap   *float64
	mipRelGap   *float64
	threads     *int
	presolve    *string
	extraBool   map[string]bool
	extraInt    map[string]int
	extraFloat  map[string]float64
	extraString map[string]string
}

func defaultSolveConfig() *solveConfig {
	return &solveConfig{
		extraBool:   make(map[string]bool),
		extraInt:    make(map[string]int),
		extraFloat:  make(map[string]float64),
		extraString: make(map[string]string),
	}
}

func (c *solveConfig) apply(s *Solver) error {
	if c.output != nil {
		if err := s.SetBoolOption("output_flag", *c.output); err != nil {
			return err
		}
	}
	if c.timeLimit != nil {
		if err := s.SetFloatOption("time_limit", *c.timeLimit); err != nil {
			return err
		}
	}
	if c.mipAbsGap != nil {
		if err := s.SetFloatOption("mip_abs_gap", *c.mipAbsGap); err != nil {
			return err
		}
	}
	if c.mipRelGap != nil {
		if err := s.SetFloatOption("mip_rel_gap", *c.mipRelGap); err != nil {
			return err
		}
	}
	if c.threads != nil {
		if err := s.SetIntOption("threads", *c.threads); err != nil {
			return err
		}
	}
	if c.presolve != nil {
		if err := s.SetStringOption("presolve", *c.presolve); err != nil {
			return err
		}
	}
	for k, v := range c.extraBool {
		if err := s.SetBoolOption(k, v); err != nil {
			return err
		}
	}
	for k, v := range c.extraInt {
		if err := s.SetIntOption(k, v); err != nil {
			return err
		}
	}
	for k, v := range c.extraFloat {
		if err := s.SetFloatOption(k, v); err != nil {
			return err
		}
	}
	for k, v := range c.extraString {
		if err := s.SetStringOption(k, v); err != nil {
			return err
		}
	}
	return nil
}

// WithOutput enables or disables solver output.
func WithOutput(enabled bool) SolveOption {
	return func(c *solveConfig) {
		c.output = &enabled
	}
}

// WithTimeLimit sets the time limit in seconds.
func WithTimeLimit(seconds float64) SolveOption {
	return func(c *solveConfig) {
		c.timeLimit = &seconds
	}
}

// WithMIPAbsGap sets the absolute MIP gap tolerance.
func WithMIPAbsGap(gap float64) SolveOption {
	return func(c *solveConfig) {
		c.mipAbsGap = &gap
	}
}

// WithMIPRelGap sets the relative MIP gap tolerance.
func WithMIPRelGap(gap float64) SolveOption {
	return func(c *solveConfig) {
		c.mipRelGap = &gap
	}
}

// WithThreads sets the number of threads to use.
func WithThreads(n int) SolveOption {
	return func(c *solveConfig) {
		c.threads = &n
	}
}

// WithPresolve sets the presolve mode ("off", "choose", "on").
func WithPresolve(mode string) SolveOption {
	return func(c *solveConfig) {
		c.presolve = &mode
	}
}

// WithBoolOption sets a custom boolean option.
func WithBoolOption(name string, value bool) SolveOption {
	return func(c *solveConfig) {
		c.extraBool[name] = value
	}
}

// WithIntOption sets a custom integer option.
func WithIntOption(name string, value int) SolveOption {
	return func(c *solveConfig) {
		c.extraInt[name] = value
	}
}

// WithFloatOption sets a custom floating-point option.
func WithFloatOption(name string, value float64) SolveOption {
	return func(c *solveConfig) {
		c.extraFloat[name] = value
	}
}

// WithStringOption sets a custom string option.
func WithStringOption(name, value string) SolveOption {
	return func(c *solveConfig) {
		c.extraString[name] = value
	}
}
