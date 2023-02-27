# 指定 Go 镜像
FROM golang:1.17-alpine
LABEL maintainer="<embiid99@foxmail.com>"

# 定义工作目录
WORKDIR /app

# 设置环境变量
ENV BUILD_ENV local

# 暴露端口
EXPOSE 6399

# 复制应用程序到工作目录
ADD . .

RUN ls

# 编译应用程序
RUN go build -o slava main.go

# RUN go build -ldflags "-w -X main.version=${VERSION}" -o /nats

# 运行应用程序
CMD ["./slava"]
