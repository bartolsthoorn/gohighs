# Dockerfile to test gohighs package with static libraries
#
# Build:
#   docker build -t gohighs-test .
#
# Run:
#   docker run --rm gohighs-test

FROM golang:1.25.5-trixie AS builder

# Install build dependencies for CGO
RUN apt-get update && apt-get install -y \
	gcc \
	g++ \
	libc6-dev \
	zlib1g-dev \
	&& rm -rf /var/lib/apt/lists/*

# Copy the gohighs module
WORKDIR /gohighs
COPY . .

# Create a test project that uses gohighs as a dependency
WORKDIR /app

# Create go.mod for test project
RUN cat > go.mod << 'EOF'
module testapp

go 1.25.5

require github.com/bartolsthoorn/gohighs v0.0.0

replace github.com/bartolsthoorn/gohighs => /gohighs
EOF

# Create a simple test program
RUN cat > main.go << 'EOF'
package main

import (
"fmt"
"log"
"math"

"github.com/bartolsthoorn/gohighs/highs"
)

func main() {
fmt.Println("Testing gohighs with static library...")

// Simple LP: Minimize x + y subject to x + y >= 1, 0 <= x,y <= 10
model := highs.Model{
ColCosts: []float64{1.0, 1.0},
ColLower: []float64{0.0, 0.0},
ColUpper: []float64{10.0, 10.0},
}
model.AddDenseRow(1.0, []float64{1.0, 1.0}, math.Inf(1)) // x + y >= 1

solution, err := model.Solve(highs.WithOutput(false))
if err != nil {
log.Fatalf("Solve failed: %v", err)
}

if !solution.IsOptimal() {
log.Fatalf("Solution not optimal: %s", solution.Status)
}

fmt.Printf("✓ Solution found!\n")
fmt.Printf("  x = %.2f, y = %.2f\n", solution.ColValues[0], solution.ColValues[1])
fmt.Printf("  Objective = %.2f\n", solution.Objective)

// Test low-level API
solver, err := highs.NewSolver()
if err != nil {
log.Fatalf("NewSolver failed: %v", err)
}
defer solver.Close()

fmt.Println("✓ Low-level API works!")
fmt.Println("✓ All tests passed - static library linking successful!")
}
EOF

# Build the test application
RUN CGO_ENABLED=1 go build -v -o testapp .

# Verify the binary is statically linked (no libhighs.so dependency)
RUN echo "Checking library dependencies:" && ldd testapp || true

# Final stage - minimal image to run the test
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y \
	libstdc++6 \
	zlib1g \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/testapp /testapp

CMD ["/testapp"]
