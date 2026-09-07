# mxd-login-shell

CMS079（盛大/国服 079）**Go 登录壳**：自己画登录 / 选区 / 选角，选完后拉起几乎原版客户端进游戏。

对接服务端 fork：[lnatpunblhna/MapleStory](https://github.com/lnatpunblhna/MapleStory)（aoaostar 系）。

## 架构

```text
[本 Go 壳 UI] --HTTP 127.0.0.1--> [Java LoginBridge] --putLoginAuth--> [Channel]
       |
       +---- 拉起 MapleStory.exe（指向频道 IP/端口）
```

- 不做完整游戏渲染（老客户端负责）。
- 交接默认走服端本机 HTTP 桥；后续可再换成纯协议垫片。

## 状态

骨架：bridge 客户端 + `cmd/shell` 健康检查。UI 与拉起客户端下一步再做。

## 本地跑

```bash
go run ./cmd/shell -bridge http://127.0.0.1:17979
```

需先启动 MapleStory 的 LoginBridge（见服务端 README）。

## 相关

- 服务端：`lnatpunblhna/MapleStory`
- 药量观测（无关登录）：`lnatpunblhna/mxd-ggv`
