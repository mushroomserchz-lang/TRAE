#!/usr/bin/env bash
# Trae CN 每日自动签到
# 原理：无头 X 桌面(Xvfb+openbox) 启动/复用 Trae 客户端，通过自定义快捷键
#       ctrl+alt+shift+k -> workbench.action.icubeClaimCheckinCredits 触发签到，
#       再用客户端日志确认 claim 接口返回 200。
set -u

DIR="$(cd "$(dirname "$0")" && pwd)"
DISPLAY_NUM=":98"
SCREEN="1440x900x24"
LOG="$DIR/logs/checkin.log"
SHOTS="$DIR/shots"
APP_USER="trae"
APP_HOME="/home/trae"
APP_BIN="/usr/share/trae-cn/trae-cn"
PROXY="${HTTPS_PROXY:-${https_proxy:-http://127.0.0.1:18080}}"
KEY="ctrl+alt+shift+k"

mkdir -p "$DIR/logs" "$SHOTS"
log() { echo "[$(date '+%F %T')] $*" | tee -a "$LOG"; }

# ---------- 1. 虚拟桌面 ----------
if ! DISPLAY=$DISPLAY_NUM xdpyinfo >/dev/null 2>&1; then
  log "启动 Xvfb $DISPLAY_NUM"
  setsid Xvfb "$DISPLAY_NUM" -screen 0 "$SCREEN" -nolisten tcp </dev/null >>"$DIR/logs/xvfb.log" 2>&1 &
  sleep 2
fi
if ! pgrep -u root -x openbox >/dev/null 2>&1; then
  log "启动 openbox"
  DISPLAY=$DISPLAY_NUM setsid openbox </dev/null >>"$DIR/logs/openbox.log" 2>&1 &
  sleep 2
fi

# ---------- 2. 启动/复用客户端 ----------
if ! pgrep -u "$APP_USER" -f "^$APP_BIN" >/dev/null 2>&1; then
  log "启动 Trae 客户端"
  sudo -u "$APP_USER" env HOME="$APP_HOME" DISPLAY="$DISPLAY_NUM" \
    HTTP_PROXY="$PROXY" HTTPS_PROXY="$PROXY" ALL_PROXY="$PROXY" \
    http_proxy="$PROXY" https_proxy="$PROXY" NO_PROXY="localhost,127.0.0.1" \
    setsid dbus-run-session -- "$APP_BIN" --no-sandbox --disable-gpu \
    --disable-dev-shm-usage --proxy-server="$PROXY" </dev/null \
    >>"$DIR/logs/app.log" 2>&1 &
fi

# 等待主窗口
for i in $(seq 1 60); do
  if DISPLAY=$DISPLAY_NUM xdotool search --name "Trae CN" >/dev/null 2>&1; then break; fi
  sleep 2
done
DISPLAY=$DISPLAY_NUM xdotool search --name "Trae CN" >/dev/null 2>&1 \
  || { log "错误：Trae 窗口未出现"; exit 1; }

WIN=$(DISPLAY=$DISPLAY_NUM xdotool search --name "Trae CN" | tail -1)
DISPLAY=$DISPLAY_NUM xdotool windowactivate --sync "$WIN" 2>/dev/null
sleep 3

# ---------- 3. 触发签到（最多 3 次，按新增日志判定结果） ----------
LOGDIR=$(ls -d "$APP_HOME/.config/Trae CN/logs/"2026* 2>/dev/null | tail -1)
LOGFILE="$LOGDIR/main.log"
result=1
for attempt in 1 2 3; do
  off=$(stat -c%s "$LOGFILE" 2>/dev/null || echo 0)
  log "第 $attempt 次触发签到"
  # 先点击编辑区获取焦点，再触发快捷键（首次按键偶发不生效）
  DISPLAY=$DISPLAY_NUM xdotool mousemove 900 300 click 1
  sleep 1
  DISPLAY=$DISPLAY_NUM xdotool key --clearmodifiers "$KEY"
  sleep 2
  DISPLAY=$DISPLAY_NUM xdotool key --clearmodifiers "$KEY"
  sleep 15

  newlog=$(tail -c +"$((off + 1))" "$LOGFILE" 2>/dev/null)
  if echo "$newlog" | grep -q "checkin_credits/claim 200 OK"; then
    log "签到成功（claim 接口 200 OK）"
    result=0; break
  elif echo "$newlog" | grep -q "checkin_credits/status 200 OK"; then
    log "今日已签到（签到命令已执行，状态查询正常）"
    result=0; break
  fi

  log "第 $attempt 次未确认，切换到 IDE 工作台后重试"
  DISPLAY=$DISPLAY_NUM xdotool mousemove 1336 87 click 1
  sleep 8
done

DISPLAY=$DISPLAY_NUM import -window root "$SHOTS/checkin-$(date '+%F').png" 2>/dev/null

if [ $result -eq 0 ]; then
  log "完成"
  exit 0
else
  log "失败：未检测到签到成功日志"
  exit 2
fi
