# mxd-login-shell

CMS079（盛大/国服 079）**Go 登录壳**：登录 / 选区 / 选角后，交接给几乎原版客户端进游戏。

对接服务端：[lnatpunblhna/MapleStory](https://github.com/lnatpunblhna/MapleStory) 分支 `feat/login-bridge`。

## 架构

```text
[本 Go 壳] --HTTP 127.0.0.1:17979--> [Java LoginBridge] --putLoginAuth--> [Channel]
     |
     +---- 拉起 MapleStory.exe（下一步）
```

## 当前能力

CLI 走通：`/health` → `/api/login` → `/api/worlds` → `/api/characters` → `/api/select`。

尚未：图形 UI、自动拉起老客户端。

## 用法

1. 检出并重建服端 `feat/login-bridge`（见 MapleStory README）。
2. 启动 CMS079 服务端（LoginBridge 默认 `127.0.0.1:17979`）。
3. 本仓库：

```bash
go run ./cmd/shell -bridge http://127.0.0.1:17979
# 或
go run ./cmd/shell -user you -pass secret
```

选角成功后会打印频道 `host:port`，供后续启动器使用。

## 相关

- 服务端 LoginBridge：`feat/login-bridge`
- 药量观测（无关）：`mxd-ggv`
