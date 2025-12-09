#!/bin/bash
#
# Build HiGHS static library for Go bindings.
#
# Usage:
#   ./scripts/build-highs.sh [OPTIONS] [HIGHS_SOURCE_DIR]
#
# Options:
#   --platform <PLATFORM>   Target platform (auto-detected if not specified)
#                           Supported: darwin_arm64, darwin_amd64, linux_amd64, linux_arm64
#   --docker                Use Docker to build (required for Linux from macOS)
#   --all                   Build for all platforms (uses Docker for Linux)
#   --help                  Show this help message
#
# Examples:
#   ./scripts/build-highs.sh                              # Build for current platform
#   ./scripts/build-highs.sh --platform darwin_amd64      # Cross-compile for Intel Mac
#   ./scripts/build-highs.sh --platform linux_amd64 --docker  # Build Linux via Docker
#   ./scripts/build-highs.sh --all                        # Build all platforms
#   ./scripts/build-highs.sh /path/to/HiGHS              # Use local HiGHS source
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Default values
HIGHS_DIR=""
TARGET_PLATFORM=""
USE_DOCKER=false
BUILD_ALL=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --platform)
            TARGET_PLATFORM="$2"
            shift 2
            ;;
        --docker)
            USE_DOCKER=true
            shift
            ;;
        --all)
            BUILD_ALL=true
            shift
            ;;
        --help)
            head -25 "$0" | tail -22
            exit 0
            ;;
        *)
            if [[ -z "$HIGHS_DIR" ]]; then
                HIGHS_DIR="$1"
            fi
            shift
            ;;
    esac
done

# Detect current platform
detect_platform() {
    local os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    local arch="$(uname -m)"
    
    case "$os" in
        darwin) os="darwin" ;;
        linux)  os="linux" ;;
        *)      echo "Unsupported OS: $os"; exit 1 ;;
    esac
    
    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)      echo "Unsupported architecture: $arch"; exit 1 ;;
    esac
    
    echo "${os}_${arch}"
}

# Clone HiGHS if needed
ensure_highs_source() {
    if [[ -z "$HIGHS_DIR" ]]; then
        HIGHS_DIR="$(mktemp -d)/HiGHS"
        echo "Cloning HiGHS..."
        git clone --depth 1 https://github.com/ERGO-Code/HiGHS.git "$HIGHS_DIR"
    fi
    
    if [[ ! -f "$HIGHS_DIR/CMakeLists.txt" ]]; then
        echo "Error: $HIGHS_DIR does not appear to be a HiGHS source directory"
        exit 1
    fi
    
    # Make path absolute
    HIGHS_DIR="$(cd "$HIGHS_DIR" && pwd)"
}

# Build for a specific platform natively
build_native() {
    local platform="$1"
    local os="${platform%_*}"
    local arch="${platform#*_}"
    
    echo ""
    echo "========================================"
    echo "Building HiGHS for $platform (native)"
    echo "========================================"
    
    local build_dir="$HIGHS_DIR/build-$platform"
    mkdir -p "$build_dir"
    cd "$build_dir"
    
    # CMake arguments
    local cmake_args=(
        -DCMAKE_BUILD_TYPE=Release
        -DBUILD_SHARED_LIBS=OFF
        -DCMAKE_POSITION_INDEPENDENT_CODE=ON
        -DBUILD_TESTING=OFF
    )
    
    # Platform-specific flags
    if [[ "$os" == "darwin" ]]; then
        cmake_args+=(-DCMAKE_OSX_DEPLOYMENT_TARGET=11.0)
        
        # Cross-compile for different Mac architecture
        if [[ "$arch" == "amd64" ]]; then
            cmake_args+=(-DCMAKE_OSX_ARCHITECTURES=x86_64)
        elif [[ "$arch" == "arm64" ]]; then
            cmake_args+=(-DCMAKE_OSX_ARCHITECTURES=arm64)
        fi
    fi
    
    echo "Configuring CMake..."
    cmake .. "${cmake_args[@]}"
    
    echo "Building..."
    make -j"$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 4)"
    
    if [[ ! -f "lib/libhighs.a" ]]; then
        echo "Error: libhighs.a was not built"
        exit 1
    fi
    
    # Install to project
    install_files "$platform" "$build_dir"
}

# Build using Docker
build_docker() {
    local platform="$1"
    local os="${platform%_*}"
    local arch="${platform#*_}"
    
    if [[ "$os" != "linux" ]]; then
        echo "Error: Docker builds only supported for Linux targets"
        exit 1
    fi
    
    echo ""
    echo "========================================"
    echo "Building HiGHS for $platform (Docker)"
    echo "========================================"
    
    local docker_platform="linux/$arch"
    local docker_arch_flag=""
    
    if [[ "$arch" == "amd64" ]]; then
        docker_platform="linux/amd64"
    elif [[ "$arch" == "arm64" ]]; then
        docker_platform="linux/arm64"
    fi
    
    # Create a temporary build script
    local build_script=$(mktemp)
    cat > "$build_script" << 'DOCKER_BUILD_SCRIPT'
#!/bin/bash
set -e

apt-get update
apt-get install -y cmake g++ zlib1g-dev make

# Copy source to temp dir to avoid cache conflicts
cp -r /highs /tmp/highs-src
cd /tmp/highs-src
rm -rf build CMakeCache.txt

mkdir -p build && cd build

cmake .. \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=OFF \
    -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
    -DBUILD_TESTING=OFF

make -j$(nproc)

# Copy results to output
cp lib/libhighs.a /output/
cp HConfig.h /output/
DOCKER_BUILD_SCRIPT
    chmod +x "$build_script"
    
    # Create output directory
    local output_dir="$PROJECT_DIR/internal/highs/lib/$platform"
    mkdir -p "$output_dir"
    
    echo "Running Docker build for $docker_platform..."
    docker run --rm \
        --platform "$docker_platform" \
        -v "$HIGHS_DIR:/highs:ro" \
        -v "$output_dir:/output" \
        -v "$build_script:/build.sh:ro" \
        ubuntu:22.04 \
        /build.sh
    
    rm -f "$build_script"
    
    # Copy headers (same for all platforms)
    copy_headers "$platform"
    
    echo ""
    echo "Built: $output_dir/libhighs.a ($(du -h "$output_dir/libhighs.a" | cut -f1))"
}

# Install built files to project
install_files() {
    local platform="$1"
    local build_dir="$2"
    
    local lib_dir="$PROJECT_DIR/internal/highs/lib/$platform"
    mkdir -p "$lib_dir"
    
    echo "Installing libhighs.a to $lib_dir..."
    cp "$build_dir/lib/libhighs.a" "$lib_dir/"
    
    # Copy headers
    copy_headers "$platform" "$build_dir"
    
    echo ""
    echo "Built: $lib_dir/libhighs.a ($(du -h "$lib_dir/libhighs.a" | cut -f1))"
}

# Copy header files
copy_headers() {
    local platform="$1"
    local build_dir="${2:-}"
    
    local include_dir="$PROJECT_DIR/internal/highs/include"
    mkdir -p "$include_dir/util"
    mkdir -p "$include_dir/lp_data"
    
    echo "Copying headers..."
    cp "$HIGHS_DIR/highs/interfaces/highs_c_api.h" "$include_dir/"
    cp "$HIGHS_DIR/highs/util/HighsInt.h" "$include_dir/util/"
    cp "$HIGHS_DIR/highs/lp_data/HighsCallbackStruct.h" "$include_dir/lp_data/"
    
    # Copy HConfig.h from build dir or output
    if [[ -n "$build_dir" && -f "$build_dir/HConfig.h" ]]; then
        cp "$build_dir/HConfig.h" "$include_dir/"
    elif [[ -f "$PROJECT_DIR/internal/highs/lib/$platform/HConfig.h" ]]; then
        mv "$PROJECT_DIR/internal/highs/lib/$platform/HConfig.h" "$include_dir/"
    fi
    
    # Patch highs_c_api.h for cgo compatibility
    echo "Patching highs_c_api.h for cgo compatibility..."
    if [[ "$(uname)" == "Darwin" ]]; then
        sed -i '' 's/^const HighsInt kHighsIis/static const HighsInt kHighsIis/g' "$include_dir/highs_c_api.h"
    else
        sed -i 's/^const HighsInt kHighsIis/static const HighsInt kHighsIis/g' "$include_dir/highs_c_api.h"
    fi
}

# Main
main() {
    ensure_highs_source
    
    echo "HiGHS source: $HIGHS_DIR"
    echo "HiGHS version: $(grep HIGHS_VERSION "$HIGHS_DIR/Version.txt" | tr '\n' ' ')"
    echo ""
    
    local current_platform="$(detect_platform)"
    local current_os="${current_platform%_*}"
    
    if [[ "$BUILD_ALL" == true ]]; then
        # Build all platforms
        echo "Building for all platforms..."
        
        # Darwin builds (native on Mac)
        if [[ "$current_os" == "darwin" ]]; then
            build_native "darwin_arm64"
            build_native "darwin_amd64"
        fi
        
        # Linux builds (via Docker)
        build_docker "linux_amd64"
        build_docker "linux_arm64"
        
    elif [[ -n "$TARGET_PLATFORM" ]]; then
        # Build for specific platform
        local target_os="${TARGET_PLATFORM%_*}"
        
        if [[ "$USE_DOCKER" == true ]] || [[ "$target_os" == "linux" && "$current_os" != "linux" ]]; then
            build_docker "$TARGET_PLATFORM"
        else
            build_native "$TARGET_PLATFORM"
        fi
        
    else
        # Build for current platform
        build_native "$current_platform"
    fi
    
    echo ""
    echo "========================================"
    echo "Build complete!"
    echo "========================================"
    echo ""
    echo "Library status:"
    for d in "$PROJECT_DIR"/internal/highs/lib/*/; do
        local p="$(basename "$d")"
        if [[ -f "$d/libhighs.a" ]]; then
            echo "  $p: $(du -h "$d/libhighs.a" | cut -f1)"
        else
            echo "  $p: (not built)"
        fi
    done
    echo ""
    echo "To test: cd $PROJECT_DIR && go test ./highs/..."
}

main
