package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Config struct {
    Username           string `json:"username"`
    Password           string `json:"password"`
    BaseURL            string `json:"base_url"`
    SearchURL          string `json:"search_url"`
    OfflineDownloadDir string `json:"offline_download_dir"`
    TelegramToken      string `json:"telegram_token"`
}

var config Config
var globalToken string

func loadConfig() error {
    config.Username = os.Getenv("BOT_USERNAME")
    config.Password = os.Getenv("BOT_PASSWORD")
    config.BaseURL = os.Getenv("BOT_BASE_URL")
    config.SearchURL = os.Getenv("BOT_SEARCH_URL")
    config.OfflineDownloadDir = os.Getenv("BOT_OFFLINE_DOWNLOAD_DIR")
    config.TelegramToken = os.Getenv("BOT_TELEGRAM_TOKEN")

    if config.Username == "" {
        file, err := os.Open("config.json")
        if err != nil {
            return err
        }
        defer file.Close()

        decoder := json.NewDecoder(file)
        if err := decoder.Decode(&config); err != nil {
            return err
        }
    }
    return nil
}

func getMagnet(fanhao string) (string, error) {
    url := config.SearchURL + fanhao
    log.Printf("Searching code: %s...", fanhao)

    client := http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
        log.Printf("Request error: %v", err)
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("HTTP status: %d", resp.StatusCode)
    }

    body, _ := io.ReadAll(resp.Body)
    
    var data map[string]interface{}
    if err := json.Unmarshal(body, &data); err != nil {
        return "", err
    }

    dataList, ok := data["data"].([]interface{})
    if !ok || len(dataList) == 0 {
        return "", fmt.Errorf("No results found")
    }

    firstEntry := fmt.Sprintf("%v", dataList[0])
    parts := strings.Split(firstEntry, ",")
    magnet := strings.TrimSpace(strings.Trim(parts[0], "['"))
    return magnet, nil
}

func getToken() (string, error) {
    if globalToken != "" {
        return globalToken, nil
    }

    url := config.BaseURL + "api/auth/login"
    loginInfo := map[string]string{
        "username": config.Username,
        "password": config.Password,
    }

    payload, _ := json.Marshal(loginInfo)
    client := http.Client{Timeout: 10 * time.Second}
    resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    var result map[string]interface{}
    if err := json.Unmarshal(body, &result); err != nil {
        return "", err
    }

    data, ok := result["data"].(map[string]interface{})
    if !ok || data["token"] == nil {
        return "", fmt.Errorf("Login failed")
    }

    globalToken = data["token"].(string)
    return globalToken, nil
}

func addMagnet(magnet string) bool {
    token, err := getToken()
    if err != nil {
        log.Printf("Get token error: %v", err)
        return false
    }

    url := config.BaseURL + "api/fs/add_offline_download"
    data := map[string]interface{}{
        "path":          config.OfflineDownloadDir,
        "urls":          []string{magnet},
        "tool":          "storage",
        "delete_policy": "delete_on_upload_succeed",
    }

    payload, _ := json.Marshal(data)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    req.Header.Set("Authorization", token)
    req.Header.Set("Content-Type", "application/json")

    client := http.Client{Timeout: 15 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("Add magnet error: %v", err)
        return false
    }
    defer resp.Body.Close()

    io.Copy(io.Discard, resp.Body) // 读取并丢弃响应体
    return resp.StatusCode == http.StatusOK
}

func triggerListRequest() {
    token, err := getToken()
    if err != nil {
        log.Printf("Get token failed: %v", err)
        return
    }

    url := config.BaseURL + "api/fs/list"
    data := map[string]interface{}{
        "path":     config.OfflineDownloadDir,
        "password": "",
        "page":     1,
        "per_page": 0,
        "refresh":  true,
    }

    payload, _ := json.Marshal(data)
    req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    req.Header.Set("Authorization", token)
    req.Header.Set("Content-Type", "application/json;charset=utf-8")

    client := http.Client{Timeout: 10 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        log.Printf("List request failed: %v", err)
        return
    }
    defer resp.Body.Close()

    io.Copy(io.Discard, resp.Body) // 读取并丢弃响应体

    if resp.StatusCode == http.StatusOK {
        log.Println("List request succeeded (200 OK)")
    } else {
        log.Printf("List request failed with status: %d", resp.StatusCode)
    }
}

func startCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
    msg := tgbotapi.NewMessage(update.Message.Chat.ID, 
        "欢迎使用离线下载机器人！\n发送磁链或番号开始使用")
    bot.Send(msg)
}

func helpCommand(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
    msg := tgbotapi.NewMessage(update.Message.Chat.ID,
        "使用说明：\n1. 直接发送磁链\n2. 发送番号自动搜索")
    bot.Send(msg)
}

func processMessage(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
    messageText := strings.TrimSpace(update.Message.Text)

    var magnet string
    var err error

    if strings.HasPrefix(messageText, "magnet:?") {
        magnet = messageText
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "正在添加任务...")
        bot.Send(msg)
    } else {
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "正在搜索资源...")
        bot.Send(msg)

        magnet, err = getMagnet(messageText)
        if err != nil {
            msg = tgbotapi.NewMessage(update.Message.Chat.ID, "搜索失败，请检查输入")
            bot.Send(msg)
            return
        }
        msg = tgbotapi.NewMessage(update.Message.Chat.ID, "已找到资源，正在添加...")
        bot.Send(msg)
    }

    if success := addMagnet(magnet); success {
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "✅ 已成功添加下载任务")
        bot.Send(msg)

        log.Println("触发目录列表请求...")
        triggerListRequest()
    } else {
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ 添加任务失败")
        bot.Send(msg)
    }
}

func main() {
    if err := loadConfig(); err != nil {
        log.Fatalf("配置加载失败: %v", err)
    }

    bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
    if err != nil {
        log.Fatalf("Bot初始化失败: %v", err)
    }

    bot.Debug = false
    log.Printf("Logged in as %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    go func() {
        for update := range updates {
            if update.Message == nil {
                continue
            }

            if update.Message.IsCommand() {
                switch update.Message.Command() {
                case "start":
                    startCommand(bot, update)
                case "help":
                    helpCommand(bot, update)
                }
            } else {
                processMessage(bot, update)
            }
        }
    }()

    log.Println("Bot服务已启动")
    signalChan := make(chan os.Signal, 1)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
    <-signalChan
    log.Println("服务已停止")
}
