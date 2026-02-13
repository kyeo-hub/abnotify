# Abnotify

Abnotify 是一款极简、纯净、安全的 Android 即时通知推送工具。它兼容 Bark 协议，支持 iOS (APNs) 和 Android (WebSocket) 双平台推送，采用创新的保活机制确保在各种 Android 系统下都能稳定运行。

## 致谢

本项目是基于以下优秀项目开发而来：

- **[Accnotify](https://github.com/trah01/Accnotify)** - 本项目 Fork 自 Accnotify，保留了其 Android 客户端、WebSocket 推送和端到端加密等核心功能。
- **[Bark](https://github.com/finb/bark)** - iOS 推送通知工具，本项目的 iOS 推送功能基于 Bark 协议实现。
- **[bark-worker-server](https://github.com/sylingd/bark-worker-server)** - 边缘函数版 Bark 服务器，本项目服务端的 APNs 实现参考了该项目。

感谢以上项目的开发者们的开源贡献！

## 核心特性

- **双平台支持**：iOS 通过 APNs 推送，Android 通过 WebSocket 推送
- **Bark 兼容**：支持标准的 Bark 推送接口，可直接使用现有的推送脚本
- **内容加密传输**：Android 端采用 RSA+AES 加密传输，保护推送内容隐私
- **超强保活**：利用 Android 辅助功能作为保活锚点，配合独立进程守护
- **Webhook 支持**：支持 GitHub、GitLab、Docker Hub、Gitea 等 Webhook 通知
- **离线存储**：支持推送历史记录，方便随时查阅

## 快速开始

### 1. 部署服务器

使用 Docker 快速部署：

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

### 2. 安装客户端

**Android：**
1. 下载并安装 Abnotify APK
2. 打开应用，输入服务器地址
3. 开启保活设置中的辅助功能和自启动权限
4. 复制推送地址即可使用

**iOS：**
1. 在 App Store 下载 Bark 客户端
2. 将服务器地址填入 Bark 设置中
3. 即可使用 Bark 推送到您的服务器

## 推送示例

### 基本推送

```bash
curl -X POST "http://your-server:8080/push/your-device-key" \
     -H "Content-Type: application/json" \
     -d '{"title":"Hello","body":"这是一条来自 Abnotify 的消息"}'
```

### Bark 兼容格式

```bash
# GET 请求
curl "http://your-server:8080/DEVICE_KEY/标题/内容"

# 带图片
curl "http://your-server:8080/DEVICE_KEY/标题/内容?image=https://example.com/image.jpg"

# 带角标
curl "http://your-server:8080/DEVICE_KEY/标题/内容?badge=1"
```

## 环境变量配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `ABNOTIFY_HOST` | 监听地址 | `0.0.0.0` |
| `ABNOTIFY_PORT` | 监听端口 | `8080` |
| `ABNOTIFY_DB_PATH` | 数据库路径 | `./data/abnotify.db` |
| `APNS_KEY_ID` | APNs Key ID | - |
| `APNS_TEAM_ID` | APNs Team ID | - |
| `APNS_PRIVATE_KEY` | APNs 私钥 (PEM) | - |
| `APNS_PRODUCTION` | 使用生产环境 | `true` |

## 保活说明

为确保应用后台稳定运行，请完成以下设置：

1. **辅助功能**：开启 Abnotify 辅助功能服务
2. **电池优化**：将 Abnotify 加入电池优化白名单
3. **自启动**：允许 Abnotify 开机自启动（部分 ROM 需要）
4. **后台运行**：允许 Abnotify 后台活动（MIUI/HyperOS 需要）
5. **通知权限**：允许 Abnotify 发送通知

## 项目结构

```
Abnotify/
├── app/                  # Android 客户端
│   ├── src/main/
│   │   ├── java/        # Kotlin 源码
│   │   └── res/         # 资源文件
│   └── build.gradle.kts
└── server/              # Go 服务器
    ├── handler/         # 请求处理器
    ├── apns/           # APNs 客户端
    ├── model/          # 数据模型
    ├── storage/        # 数据库存储
    └── main.go         # 服务器入口
```

## 相关链接

- GitHub: https://github.com/kyeo-hub/Abnotify
- 问题反馈: https://github.com/kyeo-hub/Abnotify/issues

## 许可证

本项目继承原项目的开源许可证。
