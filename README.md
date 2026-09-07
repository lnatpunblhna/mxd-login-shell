# mxd-login-shell

CMS079（盛大/国服 079）**Go 登录壳**：登录 / 选区 / 选角后，交接给几乎原版客户端进游戏。

对接服务端：[lnatpunblhna/MapleStory](https://github.com/lnatpunblhna/MapleStory)（LoginBridge 已合入 `master` / 原 `feat/login-bridge`）。

仅用于自建私服教学；不针对官方 Nexon / 无作弊功能。

## 架构（Scheme A）

```text
[本 Go 壳] --HTTP 127.0.0.1:17979--> [Java LoginBridge]
     |                                    |
     |                              putLoginAuth(charId, authIp, …)
     |
     +-- 写 handoff.json（客户端目录旁）
     +-- 启动短命本地 shim（127.0.0.1:随机端口）  [Phase1: Hello+日志]
     +-- 拉起 MapleStory.exe 127.0.0.1 <shimPort>
            │
            └── (Phase2 TODO) shim 发加密 SERVER_IP → 客户端连真实频道
```

Stock 客户端正常启动方式是 `MapleStory.exe <ip> <loginPort>`（例如 `127.0.0.1 9595`），**自己跑登录 UI**。因此只把 exe 指到频道端口不够：频道在等 `PLAYER_LOGGEDIN`，而客户端以为自己在登录服。

Scheme A：Go 壳先 `select`（服务端已 `putLoginAuth`），再让客户端连本地假登录；假登录最终下发 `getServerIP`（opcode `0x0B`），客户端才会去真实频道。

## 当前能力（Phase 1）

- CLI：`/health` → `/api/login` → `/api/worlds` → `/api/characters` → `/api/select`
- `internal/launcher`：Windows 优先拉起 `MapleStory.exe <ip> <port>`（`-client` / `MXD_CLIENT`）
- `internal/handoff`：在客户端旁写入 `handoff.json`（host/port/charId/authIp/timestamp）
- `internal/shim`：本机 TCP 监听 + 发送 CMS079 **Hello** + 记录后续加密流量；**尚未**实现 MapleAES / 假登录包 / 加密 `SERVER_IP`

### 仍 TODO（Phase 2）

1. 移植 `MapleAESOFB`（AES-ECB OFB + funnyBytes IV）
2. 解密客户端包并应答最小登录序列（`LOGIN_STATUS` / `SERVERLIST` / `CHARLIST` 等）
3. 加密发送 `MaplePacketCreator.getServerIP`（`0x0B` + IP + port + charId）
4. 确认 `putLoginAuth` 的 **authIp** 与客户端出站 IP 一致

明文 `SERVER_IP` 拼包已在 `internal/shim/packets.go`（`BuildServerIPPlain`），待套 AES 头后即可试。

## 用法

1. 更新并**重建** MapleStory 服务端 jar（LoginBridge 需在运行中的 `maple.jar` 里；改 Java 后务必 rebuild）。
2. 启动 CMS079 服务端（LoginBridge 默认 `127.0.0.1:17979`）。
3. 本仓库（**建议在 Windows** 上跑壳，以便直接起 exe）：

```bash
# 指定客户端路径（文件或所在目录）
go run ./cmd/shell -bridge http://127.0.0.1:17979 -client "D:\Games\CMS079\MapleStory.exe"

# 或用环境变量
set MXD_CLIENT=D:\Games\CMS079\MapleStory.exe
go run ./cmd/shell -user you -pass secret
```

选角成功后会：

1. 打印频道 `host:port` 与 `authIp`
2. 写 `handoff.json` 到客户端目录
3. 启动本地 shim，并执行 `MapleStory.exe 127.0.0.1 <shimPort>`

### 其它开关

| 开关 | 含义 |
|------|------|
| `-client` | `MapleStory.exe` 路径或其目录；也可用 `MXD_CLIENT` |
| `-direct` | 不走 shim，直接对**真实频道** `host:port` 启动（库存客户端仍会进自己的登录 UI） |
| `-no-launch` | 只写 `handoff.json`，不起 shim / exe |

### Windows 说明

- 启动参数必须是：`MapleStory.exe <ip> <port>`（两参数，与私服常见启动器一致）。
- 工作目录设为 exe 所在目录，以便加载 WZ/DLL。
- 在 Linux/macOS 上壳可以跑通 HTTP 登录，但拉起 `.exe` 需要 Wine；日志会提示。

### authIp 必须匹配

`LoginBridge /api/select` 调用 `LoginServer.putLoginAuth(charId, clientIp, …)`。频道校验登录时用的是客户端**出站 IP**。本机环回一般为 `127.0.0.1`；若客户端经别的网卡/代理出去，需保证与返回的 `authIp` 一致（必要时在 select JSON 里传 `clientIp`——服务端已支持该字段）。

### 重建 jar 提醒

LoginBridge 在 Java 侧。拉过 `MapleStory` 更新或合并 bridge 后，请重新编译打包并重启服务端，否则 Go 壳会连到旧进程（无 `/api/select` 等）。

## 相关

- 服务端 LoginBridge：`src/handling/login/bridge/*`
- 角色进入频道前的原版路径：`CharLoginHandler.Character_WithoutSecondPassword` → `getServerIP`
- 握手：`LoginPacket.getHello` / `MapleAESOFB`
