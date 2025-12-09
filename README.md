# gohighs

Go bindings for [HiGHS](https://highs.dev/), a high-performance solver for linear programming (LP), mixed-integer programming (MIP), and quadratic programming (QP) problems.

## Features

- **High-level API**: Simple `Model` struct for defining optimization problems
- **Low-level API**: Direct access to HiGHS solver for advanced use cases
- **Self-contained**: Embedded static libraries—`go build` produces a single binary
- **Zero runtime dependencies**: No external HiGHS installation required
- **Cross-platform**: Supports macOS (arm64, amd64) and Linux (arm64, amd64)
- **Go-idiomatic**: Functional options, proper error handling, and clean types

## Installation

```bash
go get github.com/bartolsthoorn/gohighs/highs
```

Pre-built libraries are included for all supported platforms (macOS and Linux, arm64 and amd64).

## Quick Start

### High-Level API

```go
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
```

### Mixed-Integer Programming (MIP)

```go
model := highs.Model{
    Maximize: true,
    ColCosts: []float64{1.0, 2.0, 3.0},
    ColLower: []float64{0.0, 0.0, 0.0},
    ColUpper: []float64{10.0, 10.0, 10.0},
    VarTypes: []highs.VariableType{highs.Integer, highs.Integer, highs.Continuous},
}
model.AddDenseRow(highs.NegInf(), []float64{1.0, 1.0, 1.0}, 10.0)

solution, err := model.Solve(
    highs.WithOutput(false),
    highs.WithTimeLimit(60),
    highs.WithMIPRelGap(0.01),
)
```

### Quadratic Programming (QP)

```go
// Minimize: -x2 - 3*x3 + 0.5*(2*x1² - 2*x1*x3 + 0.2*x2² + 2*x3²)
model := highs.Model{
    ColCosts: []float64{0.0, -1.0, -3.0},
    Hessian: []highs.Nonzero{
        {Row: 0, Col: 0, Val: 2.0},
        {Row: 0, Col: 2, Val: -1.0},
        {Row: 1, Col: 1, Val: 0.2},
        {Row: 2, Col: 2, Val: 2.0},
    },
}
model.AddDenseRow(highs.NegInf(), []float64{1.0, 0.0, 1.0}, 2.0)
solution, err := model.Solve()
```

### Low-Level API

```go
solver, err := highs.NewSolver()
if err != nil {
    log.Fatal(err)
}
defer solver.Close()

solver.SetBoolOption("output_flag", false)
solver.AddVars([]float64{0.0, 0.0}, []float64{10.0, 10.0})
solver.SetColCosts([]float64{1.0, 1.0})
solver.AddRow(5.0, math.Inf(1), []int{0, 1}, []float64{1.0, 1.0})

solution, err := solver.Run()
```

## Solve Options

```go
solution, err := model.Solve(
    highs.WithOutput(false),           // Disable solver output
    highs.WithTimeLimit(60),           // 60 second time limit
    highs.WithMIPAbsGap(1.0),          // Absolute MIP gap
    highs.WithMIPRelGap(0.01),         // 1% relative MIP gap
    highs.WithThreads(4),              // Use 4 threads
    highs.WithPresolve("on"),          // Enable presolve
)
```

## Building HiGHS

Pre-built libraries are included for all platforms. To rebuild or update HiGHS, use the build script:

```bash
# Build for current platform
./scripts/build-highs.sh

# Build for current platform using local HiGHS source
./scripts/build-highs.sh /path/to/HiGHS

# Build for specific platform (cross-compile on Mac)
./scripts/build-highs.sh --platform darwin_amd64

# Build for Linux using Docker (from macOS)
./scripts/build-highs.sh --platform linux_amd64 --docker
./scripts/build-highs.sh --platform linux_arm64 --docker

# Build ALL platforms at once (uses Docker for Linux)
./scripts/build-highs.sh --all
```

The script will:
1. Clone HiGHS if no source directory provided
2. Build HiGHS as a static library
3. Copy `libhighs.a` to `internal/highs/lib/<platform>/`
4. Copy and patch header files for cgo compatibility

### Platform Support

| Platform | Build Method |
|----------|--------------|
| darwin/arm64 | Native (on Apple Silicon Mac) |
| darwin/amd64 | Native cross-compile (on any Mac) |
| linux/amd64 | Docker (from any host with Docker) |
| linux/arm64 | Docker (from any host with Docker) |

**Note**: Building Linux targets requires Docker. The script uses `ubuntu:22.04` images.

## Directory Structure

```
gohighs/
├── go.mod
├── README.md
├── scripts/
│   └── build-highs.sh          # Build script for HiGHS
├── internal/
│   └── highs/
│       ├── include/            # HiGHS C headers (required for cgo)
│       │   ├── highs_c_api.h   # Main C API with function declarations and constants
│       │   ├── HConfig.h       # Build configuration (version, feature flags)
│       │   ├── util/HighsInt.h # HighsInt type definition (int vs int64)
│       │   └── lp_data/HighsCallbackStruct.h  # Callback struct for solver events
│       └── lib/                # Pre-built static libraries
│           ├── darwin_arm64/libhighs.a
│           ├── darwin_amd64/libhighs.a
│           ├── linux_amd64/libhighs.a
│           └── linux_arm64/libhighs.a
└── highs/                      # Public Go package
    ├── cgo.go                  # Cgo bindings, types, errors, low-level Solver API
    ├── model.go                # High-level Model API
    ├── solution.go             # Solution type
    ├── utils.go                # Helper functions (CSR conversion, etc.)
    └── highs_test.go           # Tests
```

**Why headers are needed**: The C headers in `internal/highs/include/` are required at compile time for cgo to understand the HiGHS C API. They define function signatures, type definitions (like `HighsInt`), and constants (like `kHighsModelStatusOptimal`). Without these headers, Go cannot call the functions in the static library. The headers must match the version of `libhighs.a` they were built with.

## HiGHS Version

This package is built against **HiGHS v1.12.0**.

## Acknowledgments

This package was inspired by [lanl/highs](https://github.com/lanl/highs), an excellent Go interface to HiGHS developed at Los Alamos National Laboratory. Thank you to Scott Pakin and the LANL team for their work!

### How gohighs differs from lanl/highs

| Feature | gohighs | lanl/highs |
|---------|---------|------------|
| **Dependencies** | Self-contained (embedded static libs) | Requires HiGHS system installation |
| **Build** | `go build` just works | Needs `pkg-config highs` |
| **Binary** | Single static binary | Dynamically linked |
| **Platforms** | Pre-built for macOS & Linux | macOS & Linux |
| **Use case** | Easy deployment, containers, cross-compilation | When HiGHS is already installed |

Choose **gohighs** if you want zero-dependency deployment or need to ship a single binary. Choose **[lanl/highs](https://github.com/lanl/highs)** if HiGHS is already installed on your system and you prefer dynamic linking.

## License

This Go package is provided under the MIT License.

HiGHS itself is licensed under the MIT License. See [HiGHS License](https://github.com/ERGO-Code/HiGHS/blob/master/LICENSE.txt).
