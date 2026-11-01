#!/usr/bin/env bash
# 一次性准备环境：依赖、Trae CN 客户端、快捷键配置
# 幂等，可重复执行
set -u

DIR=$(cd "$(dirname "$0")" && pwd)
DEB_URL="https://lf-cdn.trae.com.cn/obj/trae-com-cn/pkg/app/releases/stable/2.3.98528/linux/TRAE_CN-linux-x64.deb"
APP_USER="trae"
APP_HOME="/home/trae"
APP_BIN="/usr/share/trae-cn/trae-cn"

need_apt=0
for p in Xvfb openbox xdotool import dbus-run-session sudo; do
  command -v "$p" >/dev/null 2>&1 || need_apt=1
done

if [ "$need_apt" = 1 ]; then
  echo "[bootstrap] 安装系统依赖"
  DEBIAN_FRONTEND=noninteractive apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    xvfb openbox xdotool imagemagick dbus-x11 sudo curl >/dev/null
fi

if [ -x "$APP_BIN" ]; then
  echo "[bootstrap] Trae 客户端已安装"
else
  echo "[bootstrap] 下载并安装 Trae CN"
  deb=/tmp/TRAE_CN-linux-x64.deb
  [ -s "$deb" ] || curl -sL -o "$deb" "$DEB_URL"
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "$deb" >/dev/null
fi

if ! id "$APP_USER" >/dev/null 2>&1; then
  echo "[bootstrap] 创建运行用户 $APP_USER"
  useradd -m -s /bin/bash "$APP_USER"
fi

# 快捷键：ctrl+alt+shift+k -> 签到命令
KB_DIR="$APP_HOME/.config/Trae CN/User"
mkdir -p "$KB_DIR"
cp -f "$DIR/keybindings.json" "$KB_DIR/keybindings.json"
chown -R "$APP_USER":"$APP_USER" "$APP_HOME/.config" 2>/dev/null

chmod +x "$DIR"/*.sh
echo "[bootstrap] 完成"
