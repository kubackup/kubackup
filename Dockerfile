# syntax = docker/dockerfile:experimental
FROM --platform=$TARGETPLATFORM alpine:latest
LABEL maintainer="kubackup <tanyi@dowell.group>"
LABEL description="Kubackup - Kubernetes Backup Solution"

# 设置环境变量
ENV LANG=C.UTF-8 \
    TZ=Asia/Shanghai

# 复制预编译的二进制文件
ARG TARGETOS
ARG TARGETARCH
ARG VERSION

# 根据目标平台复制对应的二进制文件
COPY dist/kubackup_server_${VERSION}_${TARGETOS}_${TARGETARCH} /apps/kubackup_server

# 暴露端口
EXPOSE 8012

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8012/api/ping || exit 1

# 启动命令
ENTRYPOINT ["/apps/kubackup_server"]

