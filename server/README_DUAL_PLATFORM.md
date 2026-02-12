# Accnotify - 双平台推送服务器

支持 **iOS (Bark)** 和 **Android (Accnotify)** 的统一推送服务器！

## 🎯 功能特性

- ✅ **iOS 推送**：通过 APNs 推送到 Bark iOS 客户端
- ✅ **Android 推送**：通过 WebSocket 实时推送到 Accnotify Android 客户端
- ✅ **统一 API**：使用相同的推送接口，自动识别设备类型
- ✅ **Bark 兼容**：完全兼容 Bark 推送协议
- ✅ **端到端加密**：Android 推送支持 RSA+AES 加密传输
- ✅ **离线消息**：Android 设备离线时自动存储消息，重连后推送
- ✅ **批量推送**：同时向 iOS 和 Android 设备批量推送
- ✅ **Webhook 支持**：GitHub、GitLab、Docker Hub 等

---

## 🚀 快速开始

### 1. 编译和运行

```bash
cd server
go mod tidy
go build -o accnotify-server
./accnotify-server
```

### 2. 配置 APNs（iOS 推送必需）

设置环境变量：

```bash
export APNS_KEY_ID=你的KeyID
export APNS_TEAM_ID=你的TeamID
export APNS_PRIVATE_KEY="$(cat AuthKey_XXX.p8)"
export APNS_PRODUCTION=true
```

或在 `docker-compose.yml` 中配置：

```yaml
environment:
  - APNS_KEY_ID=LH4T9V5U4R
  - APNS_TEAM_ID=5U8LBRXG3A
  - APNS_PRIVATE_KEY=-----BEGIN PRIVATE KEY-----\nMIGTAgEAMBMG...\n-----END PRIVATE KEY-----
  - APNS_PRODUCTION=true
```

### 3. Docker 部署

```bash
cd server
docker-compose up -d
```

---

## 📱 客户端使用

### iOS (Bark 客户端)

#### 设备注册

```bash
# GET 方式
curl "http://your-server:8080/register?device_token=你的APNs_Token&device_key=设备密钥"

# POST 方式
curl -X POST http://your-server:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "device_token": "你的APNs_Token",
    "device_key": "my_iphone"
  }'
# 自动识别为 iOS 设备
```

#### 推送消息

```bash
# 方式1: GET 请求
curl "http://your-server:8080/my_iphone/标题/内容"

# 方式2: POST 请求
curl -X POST http://your-server:8080/my_iphone \
  -H "Content-Type: application/json" \
  -d '{
    "title": "标题",
    "body": "内容",
    "group": "分组",
    "sound": "bell",
    "url": "https://example.com"
  }'

# 方式3: 使用 /push 前缀
curl -X POST http://your-server:8080/push/my_iphone \
  -H "Content-Type: application/json" \
  -d '{"title":"标题","body":"内容"}'
```

### Android (Accnotify 客户端)

#### 设备注册

```bash
curl -X POST http://your-server:8080/register \
  -H "Content-Type: application/json" \
  -d '{
    "device_key": "my_android",
    "public_key": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----"
  }'
# 自动识别为 Android 设备
```

#### 建立 WebSocket 连接

在 Android 客户端中：

```kotlin
val wsUrl = "ws://your-server:8080/ws?key=my_android"
// 建立 WebSocket 连接...
```

#### 推送消息

使用与 iOS 相同的 API：

```bash
curl -X POST http://your-server:8080/my_android \
  -H "Content-Type: application/json" \
  -d '{
    "title": "标题",
    "body": "内容"
  }'
```

---

## 🔌 API 文档

### 设备注册

**POST /register**

```json
{
  "device_key": "设备密钥",
  "device_type": "ios 或 android (可选，自动识别)",
  "device_token": "APNs Token (iOS)",
  "public_key": "RSA 公钥 (Android)"
}
```

**响应：**

```json
{
  "code": 200,
  "message": "success",
  "timestamp": 1234567890,
  "data": {
    "key": "设备密钥",
    "device_key": "设备密钥",
    "device_type": "ios"
  }
}
```

### 推送消息

**方式1: POST /:device_key**

```json
{
  "title": "标题",
  "body": "内容",
  "group": "分组",
  "sound": "声音",
  "url": "跳转链接",
  "icon": "图标URL",
  "badge": 1,
  "level": "time-sensitive",
  "subtitle": "副标题",
  "call": true,
  "isArchive": true,
  "image": "图片URL",
  "markdown": "Markdown内容"
}
```

**方式2: GET /:device_key/:title/:body**

```
GET /my_device/标题/内容
```

**方式3: POST /push/:device_key**

与方式1相同

---

## ⚙️ 配置项

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| ACCNOTIFY_HOST | 监听地址 | 0.0.0.0 |
| ACCNOTIFY_PORT | 监听端口 | 8080 |
| ACCNOTIFY_DB_PATH | 数据库路径 | ./data/accnotify.db |
| ACCNOTIFY_ENABLE_HTTPS | 启用 HTTPS | false |
| ACCNOTIFY_CERT_FILE | SSL 证书路径 | - |
| ACCNOTIFY_KEY_FILE | SSL 私钥路径 | - |
| APNS_KEY_ID | APNs Key ID | - |
| APNS_TEAM_ID | APNs Team ID | - |
| APNS_PRIVATE_KEY | APNs 私钥 (PEM) | - |
| APNS_PRODUCTION | APNs 生产环境 | true |

---

## 🔐 APNs 配置说明

### 获取 APNs 密钥

1. 登录 [Apple Developer](https://developer.apple.com/account)
2. 进入 "Certificates, Identifiers & Profiles"
3. 选择 "Keys" -> 点击 "+" 创建新密钥
4. 勾选 "Apple Push Notifications service (APNs)"
5. 下载 .p8 文件（只能下载一次，请妥善保管）
6. 记录 Key ID 和 Team ID

### 配置环境变量

```bash
# 从 .p8 文件读取私钥
export APNS_PRIVATE_KEY="$(cat AuthKey_XXXX.p8)"

# 或者直接设置（注意转义换行符）
export APNS_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQg...
-----END PRIVATE KEY-----"
```

---

## 📊 工作原理

### iOS 推送流程

```
推送请求 → 服务器识别为 iOS 设备 → 构建 APNs Payload
    ↓
使用 JWT 认证 → 发送到 APNs 服务器 → iOS 设备接收
```

### Android 推送流程

```
推送请求 → 服务器识别为 Android 设备 → 加密消息（可选）
    ↓
设备在线? 
    ├─ 是 → 通过 WebSocket 实时推送 → 客户端解密显示
    └─ 否 → 存储到数据库 → 设备重连后推送
```

---

## 🔧 故障排除

### iOS 推送失败

1. 检查 APNs 配置是否正确
2. 确认 Key ID、Team ID 是否匹配
3. 检查设备 token 是否有效
4. 查看服务器日志中的 APNs 错误信息

### Android 推送离线

1. 确认 WebSocket 连接已建立
2. 检查设备密钥是否正确
3. 查看 `/ws` 端点是否可访问
4. 检查防火墙是否阻止 WebSocket 连接

### 设备未找到

- 确认设备已注册
- 检查 device_key 大小写是否一致
- 查看数据库中是否有该设备记录

---

## 📝 更新日志

### v2.3.0 (2026-02-12)
- ✨ 新增 iOS 设备支持（APNs 推送）
- ✨ 完全兼容 Bark 推送协议
- ✨ 统一 iOS 和 Android 推送 API
- ✨ 自动识别设备类型
- 🐛 修复数据库迁移问题
- 📝 更新文档，支持双平台

---

## 📄 许可证

本项目采用与 Accnotify 相同的开源许可证。
