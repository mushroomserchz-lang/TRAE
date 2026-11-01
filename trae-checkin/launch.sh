#!/usr/bin/env bash
# 启动 Trae CN 无头桌面环境（Xvfb + openbox + Trae）
set -u
DIR=/workspace/trae-checkin
DISPLAY_NUM="${DISPLAY_NUM:-:99}"

# 1. 虚拟屏幕
if ! xdpyinfo -display "$DISPLAY_NUM" >/dev/null 2>&1; then
  nohup Xvfb "$DISPLAY_NUM" -screen 0 1280x800x24 -nolisten tcp >"$DIR/logs/xvfb.log" 2>&1 &
  sleep 2
fi

# 2. 窗口管理器（保证窗口可聚焦、可点击）
if ! pgrep -u root -f "openbox" >/dev/null 2>&1; then
  DISPLAY="$DISPLAY_NUM" nohup openbox >"$DIR/logs/openbox.log" 2>&1 &
  sleep 2
fi

# 3. Trae
if pgrep -u trae -f "^/usr/share/trae-cn/trae-cn" >/dev/null 2>&1; then
  echo "trae already running"
  exit 0
fi

sudo -u trae env HOME=/home/trae DISPLAY="$DISPLAY_NUM" \
  dbus-run-session -- /usr/share/trae-cn/trae-cn \
  --no-sandbox --disable-gpu --disable-dev-shm-usage \
  >>"$DIR/logs/app.log" 2>&1 &
echo "launched"
