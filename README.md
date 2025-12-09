# RatTrap 知识库管理系统

基于 Go 的 Crisp 聊天机器人 webhook 服务器，集成 Supabase 知识库管理系统。

## 功能特性

- ✅ 用户登录认证（Supabase Auth）
- ✅ 知识库文件上传（支持文本、图片、PDF、Word）
- ✅ 知识库列表展示和管理
- ✅ 知识库删除功能
- ✅ Crisp 聊天窗口集成
- ✅ 自动回复消息到 Crisp 聊天窗口
- ✅ 作为中转站代理 Supabase API

## 快速开始

### 1. 配置环境变量

复制 `.env.example` 到 `.env` 并填入配置：

```bash
cp .env.example .env
```

在 `.env` 文件中填入：

```env
# Crisp 配置
CRISP_IDENTIFIER=your-plugin-id-here
CRISP_KEY=your-plugin-key-here
PORT=8080

# Supabase 配置
SUPABASE_URL=https://vwinvkxxheuexvpvzibt.supabase.co
SUPABASE_ANON_KEY=your-supabase-anon-key
SUPABASE_SERVICE_ROLE_KEY=your-supabase-service-role-key
```

#### 获取 Crisp API 凭证

⚠️ 重要：这里需要的是 **Plugin credentials**，不是 website_id！

1. 登录 [Crisp Dashboard](https://app.crisp.chat/)
2. 点击右上角头像 → **Your Profile**
3. 左侧菜单 → **Plugins**
4. 点击 **Create a new plugin**
5. 填写基本信息后创建
6. 复制 **Plugin ID** 到 `CRISP_IDENTIFIER`
7. 复制 **Plugin Key** 到 `CRISP_KEY`

#### 获取 Supabase API Keys

1. 登录 [Supabase Dashboard](https://supabase.com/dashboard)
2. 选择你的项目：`vwinvkxxheuexvpvzibt`
3. Settings → API
4. 复制 **anon/public** key 到 `SUPABASE_ANON_KEY`
5. 复制 **service_role** key 到 `SUPABASE_SERVICE_ROLE_KEY`

### 2. 安装依赖

```bash
go mod download
```

### 3. 运行服务器

```bash
# 开发环境
set -a && source .env && set +a && go run main.go

# 生产环境
go build -o server main.go
./server
```

服务器将在 `http://localhost:8080` 启动。

### 4. 配置 Crisp Webhook

1. 进入 [Crisp Dashboard](https://app.crisp.chat/)
2. 选择你的网站
3. Settings → Advanced configuration → Web Hooks
4. 点击 "Add a Web Hook"
5. 填入你的 webhook URL：`http://your-server-ip/crisp/message`
6. 选择事件：`message:send`
7. 保存

### 5. 访问管理界面

1. 打开浏览器访问：`http://localhost:8080/login`
2. 使用 Supabase 用户凭证登录
3. 登录成功后跳转到知识库管理页面

## 页面路由

| 路径 | 说明 |
|------|------|
| `/` | 首页（Crisp 聊天演示） |
| `/login` | 用户登录页面 |
| `/knowledge` | 知识库管理页面 |

## API 接口

### 认证接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/auth/login` | 用户登录 |

### 知识库接口

| 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|---------|
| GET | `/api/knowledge` | 获取知识库列表 | 否 |
| POST | `/api/knowledge/upload` | 上传知识库文件 | ✅ |
| GET | `/api/knowledge/{id}` | 获取单个知识库 | ✅ |
| DELETE | `/api/knowledge/{id}` | 删除知识库 | ✅ |

### Webhook 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/crisp/message` | Crisp 消息 webhook |

## 知识库管理功能

### 上传知识库

支持以下文件格式：
- 📄 文本文件：`.txt`, `.md`
- 🖼️ 图片文件：`.jpg`, `.jpeg`, `.png`, `.webp`
- 📋 文档文件：`.pdf`, `.docx`（即将支持）

上传时可以设置：
- 标题
- 分类（rodent_knowledge, trap_usage, faq, general）
- 语言（zh, en, ja, es）
- 可见性（private, public）
- 标签
- 来源 URL

### 知识库列表

- 实时展示所有知识库条目
- 显示标题、类型、分类、语言、创建时间
- 图片类型支持预览
- 支持删除操作
- 自动每 30 秒刷新

### 删除知识库

- 点击删除按钮
- 确认后永久删除（不可恢复）
- 自动刷新列表

## 自定义 AI 回复

修改 `main.go` 中的 `generateAIReply` 函数来集成你的 AI 模型：

```go
func generateAIReply(userMessage string) string {
    // TODO: 调用 Supabase RAG Edge Function
    // 例如：调用 /functions/v1/rag-qa
    return "AI 生成的回复"
}
```

## 生产环境部署

### 1. 构建二进制文件

```bash
go build -o rattrap-server main.go
```

### 2. 使用 systemd 管理（Linux）

创建服务文件 `/etc/systemd/system/rattrap.service`：

```ini
[Unit]
Description=RatTrap Knowledge Management Server
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/path/to/crisp-deployment
EnvironmentFile=/path/to/crisp-deployment/.env
ExecStart=/path/to/crisp-deployment/rattrap-server
Restart=always

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable rattrap
sudo systemctl start rattrap
sudo systemctl status rattrap
```

### 3. 使用 Nginx 反向代理

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 4. 配置 HTTPS（推荐）

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

## 目录结构

```
.
├── main.go              # Go 服务器代码
├── index.html           # 首页（Crisp 演示）
├── login.html           # 登录页面
├── knowledge.html       # 知识库管理页面
├── .env.example         # 环境变量示例
├── .env                 # 环境变量配置（不提交到 Git）
├── go.mod               # Go 模块依赖
├── go.sum               # Go 依赖校验
└── README.md            # 本文件
```

## 技术栈

- **后端**: Go 1.21+
- **前端**: HTML + CSS + Vanilla JavaScript
- **数据库**: Supabase (PostgreSQL + pgvector)
- **认证**: Supabase Auth
- **聊天**: Crisp Chat Widget
- **存储**: Supabase Storage

## 常见问题

### Q: 登录失败怎么办？

A: 检查：
1. `.env` 文件中的 `SUPABASE_URL` 和 `SUPABASE_ANON_KEY` 是否正确
2. 用户邮箱和密码是否在 Supabase 中存在
3. 浏览器控制台查看具体错误信息

### Q: 知识库上传失败？

A: 检查：
1. 文件格式是否支持
2. 文件大小是否超过 10MB
3. `.env` 文件中的 `SUPABASE_SERVICE_ROLE_KEY` 是否正确
4. 网络是否能访问 Supabase

### Q: Crisp 消息窗口不显示？

A: 检查：
1. `knowledge.html` 中的 `CRISP_WEBSITE_ID` 是否正确
2. 浏览器是否启用了广告拦截器
3. 网络是否能访问 `client.crisp.chat`

## 许可证

MIT License

## 支持

如有问题，请联系技术支持或查看 [文档](https://docs.rattrap.ai)
