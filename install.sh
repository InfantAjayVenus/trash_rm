#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="trash-rm"
SYMLINK_NAME="rm"

# Detect platform
OS="$(uname -s)"
case "$OS" in
  Linux*)  PLATFORM="linux" ;;
  Darwin*) PLATFORM="darwin" ;;
  *)
    echo "Error: Unsupported platform '$OS'. Only Linux and macOS are supported." >&2
    exit 1
    ;;
esac

# Check trash backend dependency
if [ "$PLATFORM" = "linux" ]; then
  if ! command -v trash >/dev/null 2>&1; then
    echo "Warning: 'trash' command not found. Install trash-cli before using trash-rm:" >&2
    echo "  Debian/Ubuntu:  sudo apt install trash-cli" >&2
    echo "  Fedora:         sudo dnf install trash-cli" >&2
    echo "  Arch:           sudo pacman -S trash-cli" >&2
    echo ""
  fi
else
  # macOS — osascript is bundled, but check anyway
  if ! command -v osascript >/dev/null 2>&1; then
    echo "Warning: 'osascript' not found. Requires osascript (bundled with macOS) — should be available by default." >&2
    echo ""
  fi
fi

# Build the binary
echo "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" .

# Install binary
echo "Installing $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME..."
install -m 755 "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
rm -f "$BINARY_NAME"

# Create rm symlink
SYMLINK_PATH="$INSTALL_DIR/$SYMLINK_NAME"
if [ -e "$SYMLINK_PATH" ] && [ ! -L "$SYMLINK_PATH" ]; then
  echo "Error: $SYMLINK_PATH exists and is not a symlink. Remove it manually before installing." >&2
  exit 1
fi

echo "Creating symlink: $SYMLINK_PATH -> $BINARY_NAME"
ln -sf "$BINARY_NAME" "$SYMLINK_PATH"

echo ""
echo "Installed successfully."
echo "  $INSTALL_DIR/$BINARY_NAME"
echo "  $INSTALL_DIR/$SYMLINK_NAME -> $BINARY_NAME"
echo ""
echo "All 'rm' invocations will now go through trash-rm."
echo "Use 'rm --be-brave-skip-trash' to bypass trash for a single invocation."
