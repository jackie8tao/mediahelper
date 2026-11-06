#!/bin/bash
set -euo pipefail

# Check root privileges
if [ "$(id -u)" -ne 0 ]; then
    echo "[ERROR] This uninstaller must be run as root or with sudo." >&2
    exit 1
fi

echo "=== Uninstalling MediaHelper from Linux ==="

# 1. Stop and disable systemd service
if systemctl is-active --quiet mediahelper.service 2>/dev/null; then
    echo "--> Stopping mediahelper service..."
    systemctl stop mediahelper.service
fi

if systemctl is-enabled --quiet mediahelper.service 2>/dev/null; then
    echo "--> Disabling mediahelper service..."
    systemctl disable mediahelper.service
fi

# 2. Remove systemd service unit
if [ -f "/etc/systemd/system/mediahelper.service" ]; then
    echo "--> Removing systemd service file..."
    rm -f /etc/systemd/system/mediahelper.service
    systemctl daemon-reload
fi

# 3. Remove binary
if [ -f "/usr/local/bin/mediahelper" ]; then
    echo "--> Removing binary /usr/local/bin/mediahelper..."
    rm -f /usr/local/bin/mediahelper
fi

# 4. Optional cleanup for configuration and database
echo ""
echo "MediaHelper binary and system service have been uninstalled."
echo "Configuration (/etc/mediahelper) and database (/var/lib/mediahelper) were kept to protect your data."
echo "If you wish to completely wipe data and configs, run:"
echo "  sudo rm -rf /etc/mediahelper /var/lib/mediahelper"

