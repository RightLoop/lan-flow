# Lan Flow — 局域网文件流转

> 极轻量的局域网文件共享工具，打开浏览器就能用。

Lan Flow 是一个单文件、零依赖的 Windows 后台服务。在电脑上跑起来后，**同一局域网内的任何设备**（手机、平板、其他电脑）打开浏览器就能上传/下载文件、分享文字消息。不需要装任何 App，不需要登录账号，不需要云服务中转。

---

## 快速开始

### 运行

```powershell
.\lan-flow.exe serve
```

终端会打印出几个地址，像这样：

```
Lan Flow listening on 0.0.0.0:8787
URL: http://192.168.1.100:8787
URL: http://172.20.0.100:8787
URL: http://127.0.0.1:8787
```

拿其他设备的浏览器打开 `http://192.168.1.100:8787`（以你终端里实际打印的为准），就能看到页面了。

### 查看状态

```powershell
.\lan-flow.exe status
```

### 安装开机自启

```powershell
.\lan-flow.exe install-startup
```

之后每次登录 Windows 都会在后台自动启动，没有弹窗。

### 移除开机自启

```powershell
.\lan-flow.exe uninstall-startup
```

---

## 使用场景

| 场景 | 说明 |
|------|------|
| 📁 传文件给隔壁工位的同事 | 拖拽上传，对方浏览器下载，不用 U 盘不用微信 |
| 📝 分享一段文本/链接/代码 | 粘贴到消息框，对方直接复制 |
| 📱 手机和电脑互传 | 手机浏览器打开地址就能操作，不用数据线 |
| 🏠 家里设备互传 | Windows 台式机跑服务，笔记本/iPad/手机都能访问 |

---

## 数据目录

默认数据存储在 `%LOCALAPPDATA%\LanFlow`，里面包含：

```
config.json     配置（端口、访问白名单等）
runtime.json    当前运行状态
metadata.json   文件记录
messages.json   消息记录
files/          上传的文件
```

上传的文件默认保留 **24 小时**后自动清理。

---

## 安全说明

- **没有密码**（设计给可信局域网用的）
- **自动限制访问范围** — 只允许本机和私有网段（`10.x.x.x`、`172.16-31.x.x`、`192.168.x.x`）访问
- **端口回退** — 默认 `8787`，被占用时自动尝试后续端口直到 `8807`
- **实例互斥** — 同目录下只会有一个实例在运行，第二次启动会自动检测并提示
- ⚠️ **不要直接暴露到公网**

---

## 技术参数

| 项目 | 值 |
|------|-----|
| 运行环境 | Windows （Go 实现，单 exe） |
| 默认端口 | 8787（端口范围 8787-8807） |
| 单文件上限 | 2048 MB |
| 消息长度上限 | 64 KB |
| 保留时间 | 24 小时（过期自动清理） |
| 界面 | 浏览器 Web 页面 |

---

## API 概览

```
GET    /                  Web 页面
GET    /api/health        服务状态
GET    /api/items         所有文件和消息
POST   /api/files         上传文件
GET    /api/files/{id}    下载文件
DELETE /api/files/{id}    删除文件
POST   /api/messages      发送消息
GET    /api/messages      消息列表
DELETE /api/messages/{id} 删除消息
```

---

## 从源码构建

需要 Go 1.22+：

```powershell
go build -o lan-flow.exe .
```

---

## 常见问题

### 端口变成了 8788 而不是 8787

如果启动后端口不是预期的 8787，说明该端口已被占用。

**常见原因：**

1. **已有实例在运行** — 用 `.\lan-flow.exe status` 检查。如果已在运行，不要重复启动。
2. **旧版的自启任务残留** — 如果你改了二进制名称或从旧版升级，老的 `LanTrans` 自启任务可能还在后台启动旧程序。重新执行 `.\lan-flow.exe install-startup` 即可——新代码会自动清理旧任务。
3. **Windows 系统保留了端口范围** — 某些 Windows 配置（尤其是 Hyper-V / WSL2）会保留一段端口范围。用这个命令排查：
   ```powershell
   netsh int ipv4 show excludedportrange protocol=tcp
   ```
   如果 `8787` 在列表中，可以去 `%LOCALAPPDATA%\LanFlow\config.json` 里修改 `preferredPort` 避开它。

---

## 开发基线

完整的设计文档和开发规划见 [BASELINE.md](BASELINE.md)。

---

## 许可

MIT
