#!/bin/sh
# 启动 WebSocket ↔ VNC 网关
# 用法: VNC_TARGET=192.168.1.20:5900 PORT=8000 ./run.sh
set -eu
: "${VNC_TARGET:?请设置 VNC_TARGET=host:port}"
export PORT="${PORT:-8000}"
exec node "$(dirname "$0")/server.js"
