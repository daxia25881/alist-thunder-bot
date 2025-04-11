# Alist Thunder Bot

一个支持离线下载的 Telegram 机器人。

## 环境变量

需要设置以下环境变量：

```
BOT_USERNAME=your_username
BOT_PASSWORD=your_password
BOT_BASE_URL=your_base_url
BOT_SEARCH_URL=your_search_url
BOT_OFFLINE_DOWNLOAD_DIR=/path/to/download
BOT_TELEGRAM_TOKEN=your_telegram_bot_token
```

## 本地运行

1. 设置环境变量或创建 config.json
2. 运行 `go run main.go`

## Docker 部署

1. 构建镜像：
   ```bash
   docker build -t alist-thunder-bot .
   ```

2. 运行容器：
   ```bash
   docker run -d \
     -e BOT_USERNAME=your_username \
     -e BOT_PASSWORD=your_password \
     -e BOT_BASE_URL=your_base_url \
     -e BOT_SEARCH_URL=your_search_url \
     -e BOT_OFFLINE_DOWNLOAD_DIR=/path/to/download \
     -e BOT_TELEGRAM_TOKEN=your_telegram_bot_token \
     alist-thunder-bot
   ```

## Render 部署

1. 在 Render 中创建新的 Web Service
2. 连接 GitHub 仓库
3. 选择 Docker 运行时
4. 设置环境变量
5. 部署 