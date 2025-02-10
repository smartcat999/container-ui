# 前端构建阶段
FROM node:23-alpine3.21 as frontend-builder

WORKDIR /app/frontend

# 复制前端文件
COPY frontend/package*.json ./
RUN npm install

COPY frontend .
RUN npm run build

# 后端构建阶段
FROM golang:1.23-alpine3.21 as backend-builder

WORKDIR /app/backend

# 复制后端文件
COPY backend/go.* ./
RUN go mod download

COPY backend .

# 构建后端
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

# 最终阶段
FROM alpine:3.21

WORKDIR /app

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist /app/dist

# 复制后端构建产物
COPY --from=backend-builder /app/backend/main .

EXPOSE 8080

CMD ["./main"] 