# Abnotify

Abnotify 是一款极简、纯净、安全的 Android 即时通知推送工具。它兼容 Bark 协议，采用创新的保活机制确保在各种 Android 系统（如 ColorOS、HyperOS 等）下都能稳定运行。

## 核心特性

- **Bark 兼容**：支持标准的 Bark 推送接口，您可以直接使用现有的推送脚本。
- **内容加密传输**：服务器到客户端采用 RSA+AES 加密传输，保护推送内容隐私。
- **超强保活**：利用 Android 辅助功能（Accessibility Service）作为保活锚点，配合独立进程守护，实现后台"永生"，告别消息延迟。
- **Webhook 支持**：支持 GitHub、GitLab、Docker Hub、Gitea 等 Webhook 通知。
- **离线存储**：支持推送历史记录，方便随时查阅。
- **无感运行**：无需 Root，不依赖 Xposed，不修改系统，安装即可使用。

## 快速开始

### 1. 部署服务器

您可以直接使用 Docker 快速部署后端：

```bash
cd server
docker-compose up -d
```

或手动构建：

```bash
cd server
go mod download
go build -o abnotify-server
./abnotify-server
```

#### 额外说明

服务器采用 Go + SQLite 架构：

- **SQLite 存储**：存储设备密钥与 RSA 公钥的对应关系，用于设备注册和验证
- **消息处理**：消息不做持久化存储，仅通过 WebSocket 实时转发
- **加密传输**：服务器收到推送后，用设备 RSA 公钥加密内容，再通过 WebSocket 发送给客户端

**安全说明**：

- **服务器可见明文**：推送者发送的内容在服务器端是明文，这个除非用客户端到客户端之间发送，不然没有办法，但那样不就变成微信了吗....
- **设备验证**：通过 device_key 确保消息只发给指定设备，防止误发
- **加密传输**：服务器到客户端采用 RSA 加密，即使 WebSocket 被截获也无法解密
- **私钥保护**：每个设备的 RSA 私钥只存储在本地，确保只有目标设备能解密内容
- **私有服务器（推荐）**：完全可控，适合敏感信息推送
- **公共服务器**：适合一般通知推送，不建议推送敏感内容

### 2. 安装客户端

1. 下载并安装 Abnotify APK。
2. 打开应用，输入您的服务器地址。
3. **重要：** 进入"保活设置"，开启"辅助功能（保活）"和自启动权限。
4. 复制生成的推送地址，即可开始发送消息。

## 推送示例

### 基本推送

```bash
curl -X POST "http://your-server:8080/push/your-device-key" \
     -H "Content-Type: application/json" \
     -d '{"title":"Hello","body":"这是一条来自 Abnotify 的消息"}'
```

### GET 请求（兼容 Bark）

注意：GET 请求内容不加密，仅用于测试。生产环境请使用 POST + HTTPS。

```bash
curl "http://your-server:8080/push/your-device-key/标题/内容"
```

## Webhook 使用

### 通用 Webhook

适用于任意 JSON 数据推送：

```
https://your-server.com/webhook/your-device-key
```

### GitHub Webhook

GitHub 仓库设置 -> Webhooks -> 添加 webhook：

```
Payload URL: https://your-server.com/webhook/your-device-key/github
Content type: application/json
```

支持事件：Push、Ping 等

### GitLab Webhook

GitLab 项目设置 -> Webhooks：

```
URL: https://your-server.com/webhook/your-device-key/gitlab
```

### Docker Hub Webhook

Docker Hub 仓库 -> Webhooks：

```
Webhook URL: https://your-server.com/webhook/your-device_key/docker
```

### Gitea Webhook

Gitea 仓库设置 -> Webhooks：

```
Target URL: https://your-server.com/webhook/your-device-key/gitea
```

## API 文档

### POST /push/:device_key

发送推送通知到指定设备。

**请求体：**

```json
{
  "title": "通知标题",
  "body": "通知内容",
  "group": "分组名称（可选）",
  "sound": "通知声音（可选）",
  "url": "点击跳转链接（可选）",
  "badge": "角标数字（可选）"
}
```

**响应：**

```json
{
  "success": true,
  "message_id": "uuid"
}
```

### POST /register

注册设备到服务器。

**请求体：**

```json
{
  "device_key": "设备唯一标识",
  "public_key": "RSA 公钥（PEM 格式）",
  "name": "设备名称（可选）"
}
```

### GET /push/:device_key/:title/:body

兼容 Bark 的 GET 请求方式。

## 客户端功能

### 首页

- 显示设备密钥和推送地址
- 一键复制推送地址
- 实时查看 WebSocket 连接状态
- 快速访问保活设置入口

### 消息页面

- 查看所有推送历史
- 消息分组展示
- 清空历史记录
- 点击消息查看详情

### 设置页面

- **服务器配置**：添加、编辑、删除服务器地址
- **服务设置**：控制前台通知显示（关闭后依赖辅助功能保活）
- **危险区域**：重置 RSA 加密密钥
- **关于项目**：查看版本信息、检查更新、访问 GitHub

## 保活说明

为确保应用后台稳定运行，请完成以下设置：

1. **辅助功能**：开启 Abnotify 辅助功能服务
2. **电池优化**：将 Abnotify 加入电池优化白名单
3. **自启动**：允许 Abnotify 开机自启动（部分 ROM 需要）
4. **后台运行**：允许 Abnotify 后台活动（MIUI/HyperOS 需要）
5. **通知权限**：允许 Abnotify 发送通知
6. **精确闹钟**：允许精确闹钟权限（Android 12+）

## 注意事项

- 关闭前台通知后，保活能力会下降，建议仅在特殊情况下关闭
- 重置加密密钥后，旧消息将无法解密，请谨慎操作
- 服务器能见推送明文，敏感内容请使用私有服务器
- device_key 是唯一凭证，请妥善保管避免泄露
- Webhook 推送的消息会自动分组到 "webhook" 组

## 项目结构

```
Abnotify/
├── app/                  # Android 客户端
│   ├── src/main/
│   │   ├── java/        # Kotlin 源码
│   │   ├── res/         # 资源文件
│   │   └── assets/      # Lottie 动画等
│   └── build.gradle.kts # 客户端构建配置
└── server/              # Go 服务器
    ├── handler/         # 请求处理器
    ├── model/          # 数据模型
    ├── storage/        # 数据库存储
    ├── crypto/         # 加密模块
    └── main.go         # 服务器入口
```

## 相关链接

- GitHub: https://github.com/trah01/Accnotify
- 问题反馈: https://github.com/trah01/Accnotify/issues

## 致谢

Accnotify 的灵感来自于 Bark 项目，并结合了 Android 无障碍服务保活的最佳实践。
