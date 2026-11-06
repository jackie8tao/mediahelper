#!/bin/bash
set -euo pipefail

# Check root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "[ERROR] This installer must be run as root or with sudo." >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_SRC="${SCRIPT_DIR}/mediahelper"
SERVICE_SRC="${SCRIPT_DIR}/mediahelper.service"
CONFIG_SRC="${SCRIPT_DIR}/config.json"

TARGET_BIN="/usr/local/bin/mediahelper"
TARGET_CONFIG_DIR="/etc/mediahelper"
TARGET_CONFIG="${TARGET_CONFIG_DIR}/config.json"
TARGET_DATA_DIR="/var/lib/mediahelper/data"
TARGET_SERVICE="/etc/systemd/system/mediahelper.service"

echo "=== Installing MediaHelper on Linux ==="

# 1. Stop existing service if running
if systemctl is-active --quiet mediahelper.service 2>/dev/null; then
    echo "--> Stopping running mediahelper service..."
    systemctl stop mediahelper.service
fi

# 2. Install executable binary
echo "--> Installing binary to ${TARGET_BIN}..."
install -m 0755 "${BIN_SRC}" "${TARGET_BIN}"

# 3. Create configuration directory & copy default configuration if not exists
echo "--> Setting up configuration in ${TARGET_CONFIG_DIR}..."
mkdir -p "${TARGET_CONFIG_DIR}"
if [ ! -f "${TARGET_CONFIG}" ]; then
    install -m 0644 "${CONFIG_SRC}" "${TARGET_CONFIG}"
    echo "    Created initial configuration: ${TARGET_CONFIG}"
else
    echo "    Existing configuration preserved: ${TARGET_CONFIG}"
    install -m 0644 "${CONFIG_SRC}" "${TARGET_CONFIG}.default"
fi

# 4. Create data storage directory
echo "--> Creating data directory in ${TARGET_DATA_DIR}..."
mkdir -p "${TARGET_DATA_DIR}"
chmod 0755 "${TARGET_DATA_DIR}"

# 5. Install systemd service unit
if [ -d "/etc/systemd/system" ]; then
    echo "--> Installing systemd service unit to ${TARGET_SERVICE}..."
    install -m 0644 "${SERVICE_SRC}" "${TARGET_SERVICE}"
    systemctl daemon-reload
    systemctl enable mediahelper.service
    echo "--> MediaHelper service enabled."
    
    echo ""
    echo "=== Installation Completed Successfully! ==="
    echo "You can manage MediaHelper with the following commands:"
    echo "  Start service:    sudo systemctl start mediahelper"
    echo "  Check status:     sudo systemctl status mediahelper"
    echo "  View live logs:   sudo journalctl -u mediahelper -f"
    echo "  Configuration:    ${TARGET_CONFIG}"
    echo "  Web UI / API:     http://<server-ip>:9090"
else
    echo "[WARN] /etc/systemd/system not found. Service unit was not installed."
    echo "You can start MediaHelper directly using:"
    echo "  ${TARGET_BIN} -config ${TARGET_CONFIG}"
fi

