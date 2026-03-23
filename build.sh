#!/bin/bash

APP_NAME="chrome-scraper"
VERSION="1.0.0"
BUILD_DIR="dist"
LDFLAGS="-s -w"

echo "[build] Chrome Scraper v${VERSION} multi-platform build"
echo "======================================"

# Clean and recreate output directory
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

# Define build targets
declare -a TARGETS=(
    "windows/amd64"
    "windows/arm64"
    "linux/amd64"
    "linux/arm64"
    "linux/386"
    "darwin/amd64"
    "darwin/arm64"
)

SUCCESS=0
FAIL=0

for TARGET in "${TARGETS[@]}"; do
    GOOS="${TARGET%/*}"
    GOARCH="${TARGET#*/}"

    OUTPUT="${BUILD_DIR}/${APP_NAME}_${VERSION}_${GOOS}_${GOARCH}"
    if [ "${GOOS}" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi

    printf "[build] %-20s --> %s\n" "${GOOS}/${GOARCH}" "${OUTPUT}"

    CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" \
        go build -trimpath -ldflags="${LDFLAGS}" -o "${OUTPUT}" .

    if [ $? -eq 0 ]; then
        SIZE=$(du -sh "${OUTPUT}" | cut -f1)
        echo "        [OK] ${SIZE}"
        SUCCESS=$((SUCCESS + 1))
    else
        echo "        [FAIL]"
        FAIL=$((FAIL + 1))
    fi
done

echo ""
echo "======================================"
echo "[done] Success: ${SUCCESS}  Failed: ${FAIL}"
echo "[done] Output directory: ${BUILD_DIR}/"
echo ""
ls -lh "${BUILD_DIR}/"
