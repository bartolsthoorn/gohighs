//go:build (linux || darwin) && (amd64 || arm64)

// Package highs provides Go bindings for the HiGHS linear optimization solver.
//
// HiGHS is a high-performance solver for linear programming (LP),
// mixed-integer programming (MIP), and quadratic programming (QP) problems.
//
// This package embeds prebuilt static HiGHS libraries, so `go build` produces
// a self-contained binary that does not require HiGHS to be installed.
//
// # Supported Platforms
//
//   - linux/amd64
//   - linux/arm64
//   - darwin/amd64
//   - darwin/arm64
//
// # High-Level API Example
//
// The high-level API uses the Model struct to define optimization problems:
//
//	model := highs.Model{
//		ColCosts:  []float64{1.0, 1.0},
//		ColLower:  []float64{0.0, 0.0},
//		ColUpper:  []float64{10.0, 10.0},
//	}
//	model.AddDenseRow(1.0, []float64{1.0, 1.0}, 5.0) // 1 <= x + y <= 5
//
//	solution, err := model.Solve()
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println("Optimal values:", solution.ColValues)
//
// # Low-Level API Example
//
// The low-level API provides direct access to the HiGHS solver:
//
//	solver, err := highs.NewSolver()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer solver.Close()
//
//	solver.SetBoolOption("output_flag", false)
//	// ... add variables and constraints
//	solution, err := solver.Run()
package highs

/*
#cgo CFLAGS: -I${SRCDIR}/../internal/highs/include

#cgo linux,amd64 LDFLAGS: ${SRCDIR}/../internal/highs/lib/linux_amd64/libhighs.a -lstdc++ -lm -ldl -lz
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/../internal/highs/lib/linux_arm64/libhighs.a -lstdc++ -lm -ldl -lz
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/../internal/highs/lib/darwin_amd64/libhighs.a -lc++ -lz
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/../internal/highs/lib/darwin_arm64/libhighs.a -lc++ -lz

#include <stdlib.h>
#include <stdint.h>
#include "highs_c_api.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"unsafe"
)

// HighsInt is the integer type used by HiGHS (matches C's HighsInt).
type HighsInt = C.HighsInt

// ----------------------------------------------------------------------------
// Types
// ----------------------------------------------------------------------------

// VariableType specifies whether a variable is continuous, integer, etc.
type VariableType int

const (
	// Continuous indicates a continuous variable (default).
	Continuous VariableType = iota
	// Integer indicates an integer variable.
	Integer
	// SemiContinuous indicates a semi-continuous variable.
	SemiContinuous
	// SemiInteger indicates a semi-integer variable.
	SemiInteger
	// ImplicitInteger indicates an implicit integer variable.
	ImplicitInteger
)

// String returns a human-readable representation of the variable type.
func (v VariableType) String() string {
	switch v {
	case Continuous:
		return "Continuous"
	case Integer:
		return "Integer"
	case SemiContinuous:
		return "SemiContinuous"
	case SemiInteger:
		return "SemiInteger"
	case ImplicitInteger:
		return "ImplicitInteger"
	default:
		return "Unknown"
	}
}

func (v VariableType) toC() C.HighsInt {
	switch v {
	case Continuous:
		return C.kHighsVarTypeContinuous
	case Integer:
		return C.kHighsVarTypeInteger
	case SemiContinuous:
		return C.kHighsVarTypeSemiContinuous
	case SemiInteger:
		return C.kHighsVarTypeSemiInteger
	case ImplicitInteger:
		return C.kHighsVarTypeImplicitInteger
	default:
		return C.kHighsVarTypeContinuous
	}
}

// Status represents the result status of a HiGHS operation.
type Status int

const (
	// StatusError indicates the operation failed with an error.
	StatusError Status = -1
	// StatusOK indicates the operation succeeded.
	StatusOK Status = 0
	// StatusWarning indicates the operation succeeded with warnings.
	StatusWarning Status = 1
)

// String returns a human-readable representation of the status.
func (s Status) String() string {
	switch s {
	case StatusError:
		return "Error"
	case StatusOK:
		return "OK"
	case StatusWarning:
		return "Warning"
	default:
		return "Unknown"
	}
}

// ModelStatus represents the status of a solved model.
type ModelStatus int

const (
	// ModelStatusNotSet indicates the model status has not been set.
	ModelStatusNotSet ModelStatus = iota
	// ModelStatusLoadError indicates an error loading the model.
	ModelStatusLoadError
	// ModelStatusModelError indicates an error in the model.
	ModelStatusModelError
	// ModelStatusPresolveError indicates an error during presolve.
	ModelStatusPresolveError
	// ModelStatusSolveError indicates an error during solve.
	ModelStatusSolveError
	// ModelStatusPostsolveError indicates an error during postsolve.
	ModelStatusPostsolveError
	// ModelStatusModelEmpty indicates the model is empty.
	ModelStatusModelEmpty
	// ModelStatusOptimal indicates an optimal solution was found.
	ModelStatusOptimal
	// ModelStatusInfeasible indicates the model is infeasible.
	ModelStatusInfeasible
	// ModelStatusUnboundedOrInfeasible indicates the model is unbounded or infeasible.
	ModelStatusUnboundedOrInfeasible
	// ModelStatusUnbounded indicates the model is unbounded.
	ModelStatusUnbounded
	// ModelStatusObjectiveBound indicates the objective bound was reached.
	ModelStatusObjectiveBound
	// ModelStatusObjectiveTarget indicates the objective target was reached.
	ModelStatusObjectiveTarget
	// ModelStatusTimeLimit indicates the time limit was reached.
	ModelStatusTimeLimit
	// ModelStatusIterationLimit indicates the iteration limit was reached.
	ModelStatusIterationLimit
	// ModelStatusUnknown indicates an unknown status.
	ModelStatusUnknown
)

// String returns a human-readable representation of the model status.
func (s ModelStatus) String() string {
	names := []string{
		"NotSet", "LoadError", "ModelError", "PresolveError",
		"SolveError", "PostsolveError", "ModelEmpty", "Optimal",
		"Infeasible", "UnboundedOrInfeasible", "Unbounded",
		"ObjectiveBound", "ObjectiveTarget", "TimeLimit",
		"IterationLimit", "Unknown",
	}
	if int(s) >= 0 && int(s) < len(names) {
		return names[s]
	}
	return "Unknown"
}

// IsOptimal returns true if the model was solved to optimality.
func (s ModelStatus) IsOptimal() bool {
	return s == ModelStatusOptimal
}

// HasSolution returns true if the model has a valid solution.
func (s ModelStatus) HasSolution() bool {
	return s == ModelStatusOptimal ||
		s == ModelStatusObjectiveBound ||
		s == ModelStatusObjectiveTarget ||
		s == ModelStatusTimeLimit ||
		s == ModelStatusIterationLimit
}

func modelStatusFromC(status C.HighsInt) ModelStatus {
	switch status {
	case C.kHighsModelStatusNotset:
		return ModelStatusNotSet
	case C.kHighsModelStatusLoadError:
		return ModelStatusLoadError
	case C.kHighsModelStatusModelError:
		return ModelStatusModelError
	case C.kHighsModelStatusPresolveError:
		return ModelStatusPresolveError
	case C.kHighsModelStatusSolveError:
		return ModelStatusSolveError
	case C.kHighsModelStatusPostsolveError:
		return ModelStatusPostsolveError
	case C.kHighsModelStatusModelEmpty:
		return ModelStatusModelEmpty
	case C.kHighsModelStatusOptimal:
		return ModelStatusOptimal
	case C.kHighsModelStatusInfeasible:
		return ModelStatusInfeasible
	case C.kHighsModelStatusUnboundedOrInfeasible:
		return ModelStatusUnboundedOrInfeasible
	case C.kHighsModelStatusUnbounded:
		return ModelStatusUnbounded
	case C.kHighsModelStatusObjectiveBound:
		return ModelStatusObjectiveBound
	case C.kHighsModelStatusObjectiveTarget:
		return ModelStatusObjectiveTarget
	case C.kHighsModelStatusTimeLimit:
		return ModelStatusTimeLimit
	case C.kHighsModelStatusIterationLimit:
		return ModelStatusIterationLimit
	default:
		return ModelStatusUnknown
	}
}

// BasisStatus represents the basis status of a variable or constraint.
type BasisStatus int

const (
	// BasisStatusLower indicates the variable is at its lower bound.
	BasisStatusLower BasisStatus = iota
	// BasisStatusBasic indicates the variable is basic.
	BasisStatusBasic
	// BasisStatusUpper indicates the variable is at its upper bound.
	BasisStatusUpper
	// BasisStatusZero indicates the variable is free and set to zero.
	BasisStatusZero
	// BasisStatusNonbasic indicates the variable is nonbasic.
	BasisStatusNonbasic
)

// String returns a human-readable representation of the basis status.
func (s BasisStatus) String() string {
	switch s {
	case BasisStatusLower:
		return "Lower"
	case BasisStatusBasic:
		return "Basic"
	case BasisStatusUpper:
		return "Upper"
	case BasisStatusZero:
		return "Zero"
	case BasisStatusNonbasic:
		return "Nonbasic"
	default:
		return "Unknown"
	}
}

func basisStatusFromC(status C.HighsInt) BasisStatus {
	switch status {
	case C.kHighsBasisStatusLower:
		return BasisStatusLower
	case C.kHighsBasisStatusBasic:
		return BasisStatusBasic
	case C.kHighsBasisStatusUpper:
		return BasisStatusUpper
	case C.kHighsBasisStatusZero:
		return BasisStatusZero
	case C.kHighsBasisStatusNonbasic:
		return BasisStatusNonbasic
	default:
		return BasisStatusLower
	}
}

// Nonzero represents a non-zero entry in a sparse matrix.
// Row and Col are zero-indexed.
type Nonzero struct {
	Row int
	Col int
	Val float64
}

// ----------------------------------------------------------------------------
// Errors
// ----------------------------------------------------------------------------

// Error represents a HiGHS error with context about which operation failed.
type Error struct {
	Op     string // Operation that failed (e.g., "Solve", "SetOption")
	Status Status // HiGHS status code
	Msg    string // Additional context
}

func (e *Error) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("highs: %s failed: %s", e.Op, e.Msg)
	}
	return fmt.Sprintf("highs: %s failed with status %s", e.Op, e.Status)
}

// newError creates a new Error if status is not OK.
// Returns nil if status is OK or Warning.
func newError(op string, status Status) error {
	if status == StatusOK || status == StatusWarning {
		return nil
	}
	return &Error{Op: op, Status: status}
}

// newErrorMsg creates a new Error with an additional message.
func newErrorMsg(op, msg string) error {
	return &Error{Op: op, Status: StatusError, Msg: msg}
}

// ----------------------------------------------------------------------------
// Solver (Low-Level API)
// ----------------------------------------------------------------------------

// Solver provides low-level access to the HiGHS solver.
// It wraps the native HiGHS instance and provides methods for
// building and solving optimization models programmatically.
//
// Always call Close() when done to release resources:
//
//	solver, _ := NewSolver()
//	defer solver.Close()
type Solver struct {
	ptr unsafe.Pointer
}

// NewSolver creates a new HiGHS solver instance.
// Returns an error if the solver could not be created.
//
// The solver must be closed with Close() when no longer needed.
func NewSolver() (*Solver, error) {
	ptr := C.Highs_create()
	if ptr == nil {
		return nil, newErrorMsg("NewSolver", "failed to create HiGHS instance")
	}

	s := &Solver{ptr: ptr}
	runtime.SetFinalizer(s, (*Solver).Close)
	return s, nil
}

// Close releases the resources held by the solver.
// It is safe to call Close multiple times.
func (s *Solver) Close() {
	if s.ptr != nil {
		C.Highs_destroy(s.ptr)
		s.ptr = nil
	}
}

// Clear resets the solver to its initial state, clearing
// the model and resetting options to defaults.
func (s *Solver) Clear() error {
	status := Status(C.Highs_clear(s.ptr))
	return newError("Clear", status)
}

// ClearModel removes all variables and constraints but keeps options.
func (s *Solver) ClearModel() error {
	status := Status(C.Highs_clearModel(s.ptr))
	return newError("ClearModel", status)
}

// ClearSolver clears solution data but keeps the model.
func (s *Solver) ClearSolver() error {
	status := Status(C.Highs_clearSolver(s.ptr))
	return newError("ClearSolver", status)
}

// Infinity returns the value used by HiGHS to represent infinity.
func (s *Solver) Infinity() float64 {
	return float64(C.Highs_getInfinity(s.ptr))
}

// NumCol returns the number of columns (variables) in the model.
func (s *Solver) NumCol() int {
	return int(C.Highs_getNumCol(s.ptr))
}

// NumRow returns the number of rows (constraints) in the model.
func (s *Solver) NumRow() int {
	return int(C.Highs_getNumRow(s.ptr))
}

// NumNonzero returns the number of non-zero entries in the constraint matrix.
func (s *Solver) NumNonzero() int {
	return int(C.Highs_getNumNz(s.ptr))
}

// SetBoolOption sets a boolean option.
func (s *Solver) SetBoolOption(name string, value bool) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var cVal C.HighsInt
	if value {
		cVal = 1
	}
	status := Status(C.Highs_setBoolOptionValue(s.ptr, cName, cVal))
	return newError("SetBoolOption", status)
}

// SetIntOption sets an integer option.
func (s *Solver) SetIntOption(name string, value int) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	status := Status(C.Highs_setIntOptionValue(s.ptr, cName, C.HighsInt(value)))
	return newError("SetIntOption", status)
}

// SetFloatOption sets a floating-point option.
func (s *Solver) SetFloatOption(name string, value float64) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	status := Status(C.Highs_setDoubleOptionValue(s.ptr, cName, C.double(value)))
	return newError("SetFloatOption", status)
}

// SetStringOption sets a string option.
func (s *Solver) SetStringOption(name, value string) error {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))
	cVal := C.CString(value)
	defer C.free(unsafe.Pointer(cVal))

	status := Status(C.Highs_setStringOptionValue(s.ptr, cName, cVal))
	return newError("SetStringOption", status)
}

// GetBoolOption returns the value of a boolean option.
func (s *Solver) GetBoolOption(name string) (bool, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.HighsInt
	status := Status(C.Highs_getBoolOptionValue(s.ptr, cName, &val))
	if err := newError("GetBoolOption", status); err != nil {
		return false, err
	}
	return val != 0, nil
}

// GetIntOption returns the value of an integer option.
func (s *Solver) GetIntOption(name string) (int, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.HighsInt
	status := Status(C.Highs_getIntOptionValue(s.ptr, cName, &val))
	if err := newError("GetIntOption", status); err != nil {
		return 0, err
	}
	return int(val), nil
}

// GetFloatOption returns the value of a floating-point option.
func (s *Solver) GetFloatOption(name string) (float64, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.double
	status := Status(C.Highs_getDoubleOptionValue(s.ptr, cName, &val))
	if err := newError("GetFloatOption", status); err != nil {
		return 0, err
	}
	return float64(val), nil
}

// SetMaximize sets whether to maximize (true) or minimize (false).
func (s *Solver) SetMaximize(maximize bool) error {
	sense := C.kHighsObjSenseMinimize
	if maximize {
		sense = C.kHighsObjSenseMaximize
	}
	status := Status(C.Highs_changeObjectiveSense(s.ptr, C.HighsInt(sense)))
	return newError("SetMaximize", status)
}

// SetObjectiveOffset sets a constant offset for the objective function.
func (s *Solver) SetObjectiveOffset(offset float64) error {
	status := Status(C.Highs_changeObjectiveOffset(s.ptr, C.double(offset)))
	return newError("SetObjectiveOffset", status)
}

// AddVar adds a single variable with the given bounds.
func (s *Solver) AddVar(lower, upper float64) error {
	status := Status(C.Highs_addVar(s.ptr, C.double(lower), C.double(upper)))
	return newError("AddVar", status)
}

// AddVars adds multiple variables with the given bounds.
func (s *Solver) AddVars(lower, upper []float64) error {
	if len(lower) != len(upper) {
		return newErrorMsg("AddVars", "lower and upper bounds must have same length")
	}
	if len(lower) == 0 {
		return nil
	}

	status := Status(C.Highs_addVars(s.ptr,
		C.HighsInt(len(lower)),
		(*C.double)(&lower[0]),
		(*C.double)(&upper[0])))
	return newError("AddVars", status)
}

// AddRow adds a constraint with the given bounds and coefficients.
// The index and value slices define the sparse row coefficients.
func (s *Solver) AddRow(lower, upper float64, index []int, value []float64) error {
	if len(index) != len(value) {
		return newErrorMsg("AddRow", "index and value must have same length")
	}

	var pIndex *C.HighsInt
	var pValue *C.double
	if len(index) > 0 {
		cIndex := make([]C.HighsInt, len(index))
		for i, v := range index {
			cIndex[i] = C.HighsInt(v)
		}
		pIndex = &cIndex[0]
		pValue = (*C.double)(&value[0])
	}

	status := Status(C.Highs_addRow(s.ptr,
		C.double(lower), C.double(upper),
		C.HighsInt(len(index)), pIndex, pValue))
	return newError("AddRow", status)
}

// AddRows adds multiple constraints in compressed sparse row format.
func (s *Solver) AddRows(lower, upper []float64, starts, index []int, value []float64) error {
	if len(lower) != len(upper) {
		return newErrorMsg("AddRows", "lower and upper bounds must have same length")
	}
	if len(index) != len(value) {
		return newErrorMsg("AddRows", "index and value must have same length")
	}
	if len(lower) == 0 {
		return nil
	}

	cStarts := make([]C.HighsInt, len(starts))
	for i, v := range starts {
		cStarts[i] = C.HighsInt(v)
	}
	cIndex := make([]C.HighsInt, len(index))
	for i, v := range index {
		cIndex[i] = C.HighsInt(v)
	}

	var pIndex *C.HighsInt
	var pValue *C.double
	if len(index) > 0 {
		pIndex = &cIndex[0]
		pValue = (*C.double)(&value[0])
	}

	status := Status(C.Highs_addRows(s.ptr,
		C.HighsInt(len(lower)),
		(*C.double)(&lower[0]), (*C.double)(&upper[0]),
		C.HighsInt(len(value)),
		&cStarts[0], pIndex, pValue))
	return newError("AddRows", status)
}

// SetColCost sets the objective coefficient for a column.
func (s *Solver) SetColCost(col int, cost float64) error {
	status := Status(C.Highs_changeColCost(s.ptr, C.HighsInt(col), C.double(cost)))
	return newError("SetColCost", status)
}

// SetColCosts sets the objective coefficients for a range of columns.
func (s *Solver) SetColCosts(costs []float64) error {
	if len(costs) == 0 {
		return nil
	}
	status := Status(C.Highs_changeColsCostByRange(s.ptr,
		0, C.HighsInt(len(costs)-1),
		(*C.double)(&costs[0])))
	return newError("SetColCosts", status)
}

// SetColBounds sets the bounds for a column.
func (s *Solver) SetColBounds(col int, lower, upper float64) error {
	status := Status(C.Highs_changeColBounds(s.ptr,
		C.HighsInt(col), C.double(lower), C.double(upper)))
	return newError("SetColBounds", status)
}

// SetColIntegrality sets the variable type for a column.
func (s *Solver) SetColIntegrality(col int, varType VariableType) error {
	status := Status(C.Highs_changeColIntegrality(s.ptr,
		C.HighsInt(col), varType.toC()))
	return newError("SetColIntegrality", status)
}

// SetIntegrality sets the variable types for a range of columns.
func (s *Solver) SetIntegrality(varTypes []VariableType) error {
	if len(varTypes) == 0 {
		return nil
	}
	integrality := make([]C.HighsInt, len(varTypes))
	for i, vt := range varTypes {
		integrality[i] = vt.toC()
	}
	status := Status(C.Highs_changeColsIntegralityByRange(s.ptr,
		0, C.HighsInt(len(varTypes)-1),
		&integrality[0]))
	return newError("SetIntegrality", status)
}

// PassModel passes a complete model to the solver in one call.
// This is more efficient than adding variables and constraints one at a time.
func (s *Solver) PassModel(
	numCol, numRow int,
	colCost, colLower, colUpper []float64,
	rowLower, rowUpper []float64,
	aStart, aIndex []int,
	aValue []float64,
	integrality []VariableType,
	maximize bool,
	offset float64,
) error {
	// Convert to C types
	sense := C.kHighsObjSenseMinimize
	if maximize {
		sense = C.kHighsObjSenseMaximize
	}

	// Convert starts and indices
	cAStart := make([]C.HighsInt, len(aStart))
	for i, v := range aStart {
		cAStart[i] = C.HighsInt(v)
	}
	cAIndex := make([]C.HighsInt, len(aIndex))
	for i, v := range aIndex {
		cAIndex[i] = C.HighsInt(v)
	}

	// Convert integrality
	var cIntegrality []C.HighsInt
	var pIntegrality *C.HighsInt
	if len(integrality) > 0 {
		cIntegrality = make([]C.HighsInt, len(integrality))
		for i, vt := range integrality {
			cIntegrality[i] = vt.toC()
		}
		pIntegrality = &cIntegrality[0]
	}

	// Get pointers
	var pColCost, pColLower, pColUpper *C.double
	var pRowLower, pRowUpper *C.double
	var pAStart, pAIndex *C.HighsInt
	var pAValue *C.double

	if len(colCost) > 0 {
		pColCost = (*C.double)(&colCost[0])
	}
	if len(colLower) > 0 {
		pColLower = (*C.double)(&colLower[0])
	}
	if len(colUpper) > 0 {
		pColUpper = (*C.double)(&colUpper[0])
	}
	if len(rowLower) > 0 {
		pRowLower = (*C.double)(&rowLower[0])
	}
	if len(rowUpper) > 0 {
		pRowUpper = (*C.double)(&rowUpper[0])
	}
	if len(cAStart) > 0 {
		pAStart = &cAStart[0]
	}
	if len(cAIndex) > 0 {
		pAIndex = &cAIndex[0]
	}
	if len(aValue) > 0 {
		pAValue = (*C.double)(&aValue[0])
	}

	status := Status(C.Highs_passModel(s.ptr,
		C.HighsInt(numCol), C.HighsInt(numRow),
		C.HighsInt(len(aValue)), 0, // num_nz, q_num_nz
		C.kHighsMatrixFormatRowwise, C.kHighsHessianFormatTriangular,
		C.HighsInt(sense), C.double(offset),
		pColCost, pColLower, pColUpper,
		pRowLower, pRowUpper,
		pAStart, pAIndex, pAValue,
		nil, nil, nil, // Hessian pointers
		pIntegrality))
	return newError("PassModel", status)
}

// PassHessian sets the Hessian matrix for quadratic programming.
// The Hessian must be provided in upper-triangular compressed sparse column format.
func (s *Solver) PassHessian(dim int, start, index []int, value []float64) error {
	if len(index) != len(value) {
		return newErrorMsg("PassHessian", "index and value must have same length")
	}

	cStart := make([]C.HighsInt, len(start))
	for i, v := range start {
		cStart[i] = C.HighsInt(v)
	}
	cIndex := make([]C.HighsInt, len(index))
	for i, v := range index {
		cIndex[i] = C.HighsInt(v)
	}

	var pStart, pIndex *C.HighsInt
	var pValue *C.double
	if len(cStart) > 0 {
		pStart = &cStart[0]
	}
	if len(cIndex) > 0 {
		pIndex = &cIndex[0]
	}
	if len(value) > 0 {
		pValue = (*C.double)(&value[0])
	}

	status := Status(C.Highs_passHessian(s.ptr,
		C.HighsInt(dim), C.HighsInt(len(value)),
		C.kHighsHessianFormatTriangular,
		pStart, pIndex, pValue))
	return newError("PassHessian", status)
}

// Run solves the model and returns the solution.
func (s *Solver) Run() (*Solution, error) {
	status := Status(C.Highs_run(s.ptr))
	if status == StatusError {
		return nil, newError("Run", status)
	}

	// Get model status
	modelStatus := modelStatusFromC(C.Highs_getModelStatus(s.ptr))

	// Get dimensions
	numCol := int(C.Highs_getNumCol(s.ptr))
	numRow := int(C.Highs_getNumRow(s.ptr))

	// Allocate solution arrays
	colValue := make([]float64, numCol)
	colDual := make([]float64, numCol)
	rowValue := make([]float64, numRow)
	rowDual := make([]float64, numRow)

	var pColValue, pColDual, pRowValue, pRowDual *C.double
	if numCol > 0 {
		pColValue = (*C.double)(&colValue[0])
		pColDual = (*C.double)(&colDual[0])
	}
	if numRow > 0 {
		pRowValue = (*C.double)(&rowValue[0])
		pRowDual = (*C.double)(&rowDual[0])
	}

	// Get solution
	C.Highs_getSolution(s.ptr, pColValue, pColDual, pRowValue, pRowDual)

	// Get objective value
	objective := float64(C.Highs_getObjectiveValue(s.ptr))

	sol := &Solution{
		Status:    modelStatus,
		ColValues: colValue,
		ColDuals:  colDual,
		RowValues: rowValue,
		RowDuals:  rowDual,
		Objective: objective,
	}

	// Try to get basis info
	if numCol > 0 && numRow > 0 {
		colBasis := make([]C.HighsInt, numCol)
		rowBasis := make([]C.HighsInt, numRow)
		basisStatus := C.Highs_getBasis(s.ptr, &colBasis[0], &rowBasis[0])
		if Status(basisStatus) == StatusOK {
			sol.ColBasis = make([]BasisStatus, numCol)
			sol.RowBasis = make([]BasisStatus, numRow)
			for i, b := range colBasis {
				sol.ColBasis[i] = basisStatusFromC(b)
			}
			for i, b := range rowBasis {
				sol.RowBasis[i] = basisStatusFromC(b)
			}
		}
	}

	return sol, nil
}

// GetIntInfo returns an integer info value.
func (s *Solver) GetIntInfo(name string) (int, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.HighsInt
	status := Status(C.Highs_getIntInfoValue(s.ptr, cName, &val))
	if err := newError("GetIntInfo", status); err != nil {
		return 0, err
	}
	return int(val), nil
}

// GetInt64Info returns a 64-bit integer info value.
func (s *Solver) GetInt64Info(name string) (int64, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.int64_t
	status := Status(C.Highs_getInt64InfoValue(s.ptr, cName, &val))
	if err := newError("GetInt64Info", status); err != nil {
		return 0, err
	}
	return int64(val), nil
}

// GetFloatInfo returns a floating-point info value.
func (s *Solver) GetFloatInfo(name string) (float64, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	var val C.double
	status := Status(C.Highs_getDoubleInfoValue(s.ptr, cName, &val))
	if err := newError("GetFloatInfo", status); err != nil {
		return 0, err
	}
	return float64(val), nil
}

// ReadModel reads a model from a file (LP, MPS, or other supported format).
func (s *Solver) ReadModel(filename string) error {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	status := Status(C.Highs_readModel(s.ptr, cFilename))
	return newError("ReadModel", status)
}

// WriteModel writes the model to a file.
func (s *Solver) WriteModel(filename string) error {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	status := Status(C.Highs_writeModel(s.ptr, cFilename))
	return newError("WriteModel", status)
}

// WriteSolution writes the solution to a file.
func (s *Solver) WriteSolution(filename string, pretty bool) error {
	cFilename := C.CString(filename)
	defer C.free(unsafe.Pointer(cFilename))

	var status C.HighsInt
	if pretty {
		status = C.Highs_writeSolutionPretty(s.ptr, cFilename)
	} else {
		status = C.Highs_writeSolution(s.ptr, cFilename)
	}
	return newError("WriteSolution", Status(status))
}

