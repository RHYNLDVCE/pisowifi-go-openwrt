#!/bin/bash

# PisoWifi System Installer Script
# This script installs all necessary dependencies for the PisoWifi Golang backend and React frontend.

# Check for root privileges
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root (e.g. sudo ./install.sh)"
  exit 1
fi

echo "=========================================="
echo "    PisoWifi System Dependency Installer  "
echo "=========================================="

echo "[1/4] Updating system package list..."
apt update -y

echo "[2/4] Installing Core Utilities & MQTT Client..."
# mosquitto-clients: Required for fail_safe.sh to communicate with the OpenWrt router
# coreutils, procps, iputils-ping, sudo: Core system commands
apt install -y \
    mosquitto-clients \
    sudo \
    coreutils \
    iputils-ping \
    procps

echo "[3/4] Installing Build Tools (Go)..."
# Install Go (golang) for compiling the backend.
# Note: This uses the default distro repositories. If you need the absolute latest version,
# you may need to install Go via the official golang.org tarball.
apt install -y golang


echo "[4/4] Installing Systemd Service..."
if [ -f "pisowifi.service" ]; then
    cp pisowifi.service /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable pisowifi
    echo "pisowifi.service installed and enabled to start on boot!"
else
    echo "Warning: pisowifi.service not found in the current directory. Skipping service install."
fi

echo "=========================================="
echo " Installation Complete!"
echo "=========================================="
echo ""
echo "Next Steps:"
echo "1. Backend: Run 'go build -o pisowifi-server cmd/server/main.go' to compile."
echo "2. Service: Run 'sudo systemctl start pisowifi' to start the backend."
echo "3. Frontend: Build your React app ('npm run build') on your PC and transfer the 'admin-ui/dist' folder."
echo ""
