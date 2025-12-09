package highs

import (
	"math"
	"sort"
)

// Inf returns positive infinity, suitable for unbounded variable bounds.
func Inf() float64 {
	return math.Inf(1)
}

// NegInf returns negative infinity, suitable for unbounded variable bounds.
func NegInf() float64 {
	return math.Inf(-1)
}

// nonzerosToCSR converts a slice of Nonzero elements to compressed sparse row format.
// If triangular is true, it validates that the matrix is upper triangular.
func nonzerosToCSR(nz []Nonzero, triangular bool) (start, index []int, value []float64, err error) {
	if len(nz) == 0 {
		return nil, nil, nil, nil
	}

	// Sort by row, then by column
	sorted := make([]Nonzero, len(nz))
	copy(sorted, nz)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Row != sorted[j].Row {
			return sorted[i].Row < sorted[j].Row
		}
		return sorted[i].Col < sorted[j].Col
	})

	// Validate and deduplicate
	filtered := make([]Nonzero, 0, len(sorted))
	for _, n := range sorted {
		if n.Row < 0 || n.Col < 0 {
			return nil, nil, nil, newErrorMsg("nonzerosToCSR", "negative row or column index")
		}
		if triangular && n.Row > n.Col {
			return nil, nil, nil, newErrorMsg("nonzerosToCSR", "Hessian must be upper triangular")
		}
		// Merge duplicates (keep last value)
		if len(filtered) > 0 && filtered[len(filtered)-1].Row == n.Row && filtered[len(filtered)-1].Col == n.Col {
			filtered[len(filtered)-1].Val = n.Val
		} else {
			filtered = append(filtered, n)
		}
	}

	// Build CSR format
	start = make([]int, 0)
	index = make([]int, len(filtered))
	value = make([]float64, len(filtered))

	prevRow := -1
	for i, n := range filtered {
		if n.Row > prevRow {
			start = append(start, i)
			prevRow = n.Row
		}
		index[i] = n.Col
		value[i] = n.Val
	}

	return start, index, value, nil
}

// expandSlice expands a slice to length n if it's empty, filling with fillValue.
// Returns the original slice if it already has length n.
// Returns an error if the slice has a non-zero length that differs from n.
func expandSlice(n int, slice []float64, fillValue float64) ([]float64, error) {
	if len(slice) == n {
		return slice, nil
	}
	if len(slice) == 0 {
		result := make([]float64, n)
		for i := range result {
			result[i] = fillValue
		}
		return result, nil
	}
	return nil, newErrorMsg("expandSlice", "inconsistent slice length")
}

// maxRowCol finds the maximum row and column indices from a slice of nonzeros.
func maxRowCol(nz []Nonzero) (maxRow, maxCol int) {
	maxRow, maxCol = -1, -1
	for _, n := range nz {
		if n.Row > maxRow {
			maxRow = n.Row
		}
		if n.Col > maxCol {
			maxCol = n.Col
		}
	}
	return maxRow, maxCol
}
