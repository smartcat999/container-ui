# Container Runtime Manager

Container Runtime Manager 是一个容器运行时管理工具，提供 Web UI 界面，支持管理本地和远程的容器运行时。目前支持 Docker 运行时。

## 功能特性

- 支持本地和远程 Docker 运行时连接管理
- 容器生命周期管理（创建、启动、停止、删除等）
- 镜像管理（拉取、构建、删除等）
- 数据卷管理
- 网络管理
- 支持 TCP 和 Unix Socket 连接方式
- 友好的 Web 操作界面

## 项目结构

```
.
├── backend                 # 后端服务
│   ├── server             # 程序入口
│   ├── internal           # 内部包
│   │   ├── config        # 配置管理
│   │   ├── handler       # HTTP 处理器
│   │   ├── middleware    # 中间件
│   │   ├── model        # 数据模型
│   │   ├── router       # 路由配置
│   │   └── service      # 业务逻辑
│   └── pkg               # 公共包
├── frontend               # 前端项目
│   ├── public            # 静态资源
│   └── src               # 源代码
│       ├── api           # API 接口
│       ├── components    # 组件
│       ├── router        # 路由配置
│       ├── store         # 状态管理
│       └── views         # 页面
├── docker-compose.yml     # Docker Compose 配置
└── Dockerfile            # Docker 构建文件
```

## 快速开始

### 使用 Docker Compose

1. 克隆仓库
```bash
git clone https://github.com/your-username/container-runtime-manager.git
cd container-runtime-manager
```

2. 启动服务
```bash
docker compose up --build
```

服务启动后，访问 http://localhost:8080 即可打开管理界面。

### 连接 Docker 运行时

1. TCP 连接
- 确保远程 Docker daemon 开启了 TCP 监听
- 在连接管理页面添加新连接，选择 TCP 方式
- 输入主机地址和端口（默认 2375）

2. Unix Socket 连接
- 默认路径为 /var/run/docker.sock
- 在连接管理页面添加新连接，选择 Socket 方式
- 输入 Socket 文件路径

## 开发环境

### 前置条件

- Go 1.19 或更高版本
- Node.js 18 或更高版本
- Docker
- Docker Compose

### 本地开发

1. 启动后端服务
```bash
go run backend/server/main.go
```

2. 启动前端开发服务
```bash
cd frontend
npm install
npm run dev
```

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。
