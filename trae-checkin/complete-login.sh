#!/usr/bin/env bash
# 完成 Trae 客户端登录：把浏览器登录后的回调地址交给客户端本地 OAuth 服务
# 用法: complete-login.sh "http://127.0.0.1:35063/authorize?code=...&state=..."
set -u
URL="${1:-}"
if [ -z "$URL" ]; then
  echo "用法: $0 '<登录后浏览器地址栏里的完整回调 URL>'" >&2
  exit 1
fi
LOG=/workspace/trae-checkin/logs/login.log
echo "[$(date '+%F %T')] callback: $URL" >>"$LOG"
curl -s -m 30 "$URL" -w "\n[curl] http=%{http_code}\n" >>"$LOG" 2>&1
tail -5 "$LOG"
