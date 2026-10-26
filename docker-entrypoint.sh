#!/bin/sh
set -e

# If running as root (e.g. initial container start with volume mount)
if [ "$(id -u)" = "0" ]; then
    mkdir -p /home/appuser/.checkIndex
    # Ensure appuser owns ~/.checkIndex so configuration can be read and written safely
    chown -R appuser:appgroup /home/appuser/.checkIndex 2>/dev/null || true
    chmod 700 /home/appuser/.checkIndex 2>/dev/null || true
    if [ -f /home/appuser/.checkIndex/config.json ]; then
        chmod 600 /home/appuser/.checkIndex/config.json 2>/dev/null || true
    fi
    exec su-exec appuser "$@"
else
    mkdir -p /home/appuser/.checkIndex 2>/dev/null || true
    exec "$@"
fi

