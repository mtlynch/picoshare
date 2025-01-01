#!/usr/bin/env bash

set -eux

# Create required directories
mkdir -p ${HOME}/.vnc

# Set up VNC password
echo "password" | vncpasswd -f > ${HOME}/.vnc/passwd
chmod 600 ${HOME}/.vnc/passwd

# Create xstartup file
cat > ${HOME}/.vnc/xstartup << EOF
#!/bin/bash
/usr/bin/startlxde
EOF

# Kill any existing VNC sessions
vncserver -kill :1 || true

# Start VNC server with logging
vncserver :1 -geometry 1280x800 -depth 24 -localhost no

# Start noVNC with logging
websockify --web=/usr/share/novnc 6080 localhost:5901 &

# Keep container running and show logs
tail -f ${HOME}/.vnc/*log
