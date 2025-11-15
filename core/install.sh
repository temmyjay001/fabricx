#!/bin/bash
# install.sh - FabricX Runtime installer

set -e

VERSION="${FABRICX_VERSION:-0.1.0}"
INSTALL_DIR="${FABRICX_INSTALL_DIR:-/usr/local/bin}"
REPO="temmyjay001/fabricx"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🚀 FabricX Runtime Installer v${VERSION}"
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo "📋 Detected system: ${OS}/${ARCH}"

# Check Docker
echo "🐳 Checking Docker..."
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed${NC}"
    echo ""
    echo "Please install Docker first:"
    echo "  • macOS: https://docs.docker.com/desktop/mac/install/"
    echo "  • Linux: https://docs.docker.com/engine/install/"
    echo "  • Windows: https://docs.docker.com/desktop/windows/install/"
    exit 1
fi

if ! docker ps &> /dev/null; then
    echo -e "${RED}❌ Docker is not running${NC}"
    echo "Please start Docker and try again"
    exit 1
fi

echo -e "${GREEN}✓ Docker is available${NC}"

# Download binary
BINARY_NAME="fabricx-runtime-${VERSION}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME="${BINARY_NAME}.zip"
    EXT=".zip"
else
    BINARY_NAME="${BINARY_NAME}.tar.gz"
    EXT=".tar.gz"
fi

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}"

echo ""
echo "📥 Downloading FabricX Runtime..."
echo "   URL: ${DOWNLOAD_URL}"

TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

if ! curl -fsSL "${DOWNLOAD_URL}" -o "${TEMP_DIR}/${BINARY_NAME}"; then
    echo -e "${RED}❌ Failed to download binary${NC}"
    echo "Please check if version ${VERSION} exists"
    exit 1
fi

# Extract
echo "📦 Extracting..."
cd "${TEMP_DIR}"
if [ "$EXT" = ".tar.gz" ]; then
    tar -xzf "${BINARY_NAME}"
else
    unzip -q "${BINARY_NAME}"
fi

# Find the binary
BINARY_FILE=$(find . -name "fabricx-runtime-*" -type f ! -name "*.tar.gz" ! -name "*.zip" | head -1)

if [ -z "$BINARY_FILE" ]; then
    echo -e "${RED}❌ Could not find binary in archive${NC}"
    exit 1
fi

# Install
echo "📂 Installing to ${INSTALL_DIR}..."
if [ -w "${INSTALL_DIR}" ]; then
    cp "${BINARY_FILE}" "${INSTALL_DIR}/fabricx-runtime"
    chmod +x "${INSTALL_DIR}/fabricx-runtime"
else
    echo "   (requires sudo)"
    sudo cp "${BINARY_FILE}" "${INSTALL_DIR}/fabricx-runtime"
    sudo chmod +x "${INSTALL_DIR}/fabricx-runtime"
fi

# Verify
if command -v fabricx-runtime &> /dev/null; then
    VERSION_OUTPUT=$(fabricx-runtime --version)
    echo ""
    echo -e "${GREEN}✅ Installation successful!${NC}"
    echo ""
    echo "${VERSION_OUTPUT}"
    echo ""
    echo "🎉 You can now start the runtime:"
    echo "   fabricx-runtime"
    echo ""
    echo "Or run in background:"
    echo "   fabricx-runtime &"
else
    echo -e "${YELLOW}⚠️  Installation completed but binary not in PATH${NC}"
    echo "   You can run it directly: ${INSTALL_DIR}/fabricx-runtime"
fi