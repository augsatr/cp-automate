#!/bin/bash

EXECUTABLE_NAME="raga"
INSTALL_DIR="$HOME/cp-automate"
EXECUTABLE_PATH="$INSTALL_DIR/$EXECUTABLE_NAME"
TEMPLATES_DIR="$INSTALL_DIR/templates"
ZSHRC="$HOME/.zshrc"

echo "Compiling the Go project..."
go build -o "$EXECUTABLE_NAME" .

if [ ! -f "$EXECUTABLE_NAME" ]; then
    echo "Error: Failed to compile the Go project."
    exit 1
fi

if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory: $INSTALL_DIR"
    mkdir -p "$INSTALL_DIR"
fi

echo "Moving $EXECUTABLE_NAME to $INSTALL_DIR"
mv "$EXECUTABLE_NAME" "$INSTALL_DIR/"

if [ -d "./templates" ]; then
    echo "Moving templates/ directory to $INSTALL_DIR"
    cp -r ./templates "$INSTALL_DIR/"
else
    echo "Warning: No templates directory found."
fi

if ! grep -q "$INSTALL_DIR" "$ZSHRC"; then
    echo "Adding $INSTALL_DIR to the PATH in $ZSHRC"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$ZSHRC"
else
    echo "$INSTALL_DIR is already in the PATH."
fi

echo "Reload the terminal"
echo "Or do export PATH=\"\$PATH:$INSTALL_DIR\""

echo "Verifying the installation..."
if command -v "$EXECUTABLE_NAME" &> /dev/null; then
    echo "$EXECUTABLE_NAME is successfully installed and accessible from the PATH!"
else
    echo "Error: $EXECUTABLE_NAME is not found in the PATH."
    exit 1
fi
