# Crisp AI 客服集成

基于 Go 的 Crisp 聊天机器人 webhook 服务器。

## 功能

- 网页端集成 Crisp 聊天窗口
- 接收用户消息的 webhook
- 自动回复消息到 Crisp 聊天窗口

## 快速开始

### 1. 配置环境变量

复制 `.env.example` 到 `.env` 并填入你的 Crisp API 凭证：

```bash
cp .env.example .env
```

在 `.env` 文件中填入：
```
CRISP_IDENTIFIER=your-identifier-here
CRISP_KEY=your-key-here
```

**获取 Crisp API 凭证：**

重要：这里需要的是 **Plugin credentials**，不是 website_id！

1. 登录 [Crisp Dashboard](https://app.crisp.chat/)
2. 点击右上角头像 → **Your Profile**（不是选择网站）
3. 左侧菜单 → **Plugins**
4. 点击 **Create a new plugin**
5. 填写基本信息：
   - Name: 随便填（例如：My Bot）
   - Description: 随便填
   - Website: 留空
6. 创建后会显示：
   - **Plugin ID** → 这个是你的 `CRISP_IDENTIFIER`
   - **Plugin Key** → 这个是你的 `CRISP_KEY`
7. 复制这两个值到 `.env` 文件

注意：
- Plugin ID 格式类似：`7f3e7b5e-xxxx-xxxx-xxxx-xxxxxxxxxxxx`
- Plugin Key 是一长串字符串
- 不要和 website_id 混淆（website_id 在代码中已经自动获取）

### 2. 运行服务器

```bash
# 加载环境变量并运行
set -a && source .env && set +a && go run main.go
```

服务器将在 `http://localhost:8080` 启动（或 `.env` 中配置的端口）。

### 3. 配置 Crisp Webhook

1. 进入 [Crisp Dashboard](https://app.crisp.chat/)
2. 选择你的网站
3. 左侧菜单 → Settings → Advanced configuration → Web Hooks
4. 点击 "Add a Web Hook"
5. 填入你的 webhook URL：`http://115.190.197.188/crisp/message`
6. 选择事件：`message:send`（用户发送消息）
7. 保存

### 4. 测试

访问 `http://localhost:8080`，你会看到网页右下角的 Crisp 聊天图标。
发送一条消息，服务器会自动回复。

## 自定义 AI 回复

修改 `main.go` 中的 `generateAIReply` 函数来集成你自己的 AI 模型：

```go
func generateAIReply(userMessage string) string {
    // 调用你的 AI API
    // 例如：OpenAI, Claude, 本地模型等
    return "AI 生成的回复"
}
```

## 生产环境部署

确保：
1. 使用 HTTPS（Crisp webhook 可能要求 HTTPS）
2. 设置正确的环境变量
3. 使用进程管理器（如 systemd, supervisor 等）
4. 配置防火墙规则

## 目录结构

```
.
├── main.go           # Go 服务器代码
├── index.html        # 前端页面
├── .env.example      # 环境变量示例
└── README.md         # 本文件
```
