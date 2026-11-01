# trae-checkin

Trae CN 桌面客户端的每日自动签到脚本（无头 Linux 沙箱版）。

## 原理

1. 在 Xvfb 虚拟桌面 + openbox 中启动 Trae CN 客户端；
2. 通过自定义快捷键 `ctrl+alt+shift+k` → `workbench.action.icubeClaimCheckinCredits` 触发签到；
3. 读取客户端日志中 `checkin_credits/claim` 接口结果确认是否签到成功。

## 用法

```bash
git clone https://github.com/mushroomserchz-lang/trae-checkin.git
cd trae-checkin
bash bootstrap.sh     # 仅首次需要：装依赖、装 Trae、写快捷键
bash checkin.sh       # 每日执行这一条即可
```

`checkin.sh` 会自动处理：虚拟桌面未启动→拉起、客户端未运行→拉起、未登录→报错退出、
签到失败→自动重试 3 次。日志在 `logs/checkin.log`，截图在 `shots/`。

## 前提：登录态

脚本复用 `~/.config/Trae CN` 里已保存的登录态。沙箱重建或登录失效时需要重新授权：

1. 启动客户端后从日志里取出 `authorization?...` 登录链接；
2. 在浏览器打开并登录 Trae；
3. 登录后浏览器会跳转到一个打不开的 `http://127.0.0.1:35063/authorize?...`，
   把该完整 URL 作为参数传给 `complete-login.sh`。

```bash
bash complete-login.sh 'http://127.0.0.1:35063/authorize?code=...'
```

## 文件

| 文件 | 作用 |
| --- | --- |
| `bootstrap.sh` | 安装依赖 / Trae 客户端 / 快捷键配置 |
| `checkin.sh` | 每日签到主脚本 |
| `complete-login.sh` | 用浏览器回调完成客户端登录 |
| `keybindings.json` | 签到命令的快捷键绑定 |
