#!/bin/bash

OS=$(uname -s)
ARCH=$(uname -m)

bin_dir="/root/data/nemu/nemu"

# 函数：安装软件包
install() {
    # 检查是否为 root 用户，非 root 用户在 Linux 和 FreeBSD 上不执行包管理器安装
    if [ "$OS" != "Darwin" ] && [ "$EUID" -ne 0 ]; then
        echo "Running in rootless mode, skipping system package installation."
        echo "Please ensure curl, wget, and sed are installed manually if needed."
        return 0
    fi

    if [ $# -eq 0 ]; then
        echo "Error: No packages specified"
        return 1
    fi

    for package in "$@"; do
        # 检查软件包是否已安装
        if ! command -v "$package" &>/dev/null; then
            echo "Installing dependency: $package"
            if command -v dnf &>/dev/null; then
                dnf -y update && dnf install -y "$package"
            elif command -v yum &>/dev/null; then
                yum -y update && yum -y install "$package"
            elif command -v apt &>/dev/null; then
                apt update -y && apt install -y "$package"
            elif command -v apk &>/dev/null; then
                apk update && apk add "$package"
            elif command -v brew &>/dev/null; then
                brew install "$package"
            elif command -v pkg &>/dev/null; then
                pkg update && pkg install -y "$package"
            else
                echo "Error: Unknown package manager. Please install $package manually"
                # 在 rootless 模式下不退出，只提示
                if [ "$EUID" -ne 0 ] && [ "$OS" != "Darwin" ]; then
                     echo "Running in rootless mode, cannot install $package. Please install manually."
                else
                    return 1
                fi
            fi
        fi
    done

    return 0
}

# 在 Linux 和 FreeBSD 上，如果是非 root 用户，不强制要求 root 权限，进入 rootless 模式
if [ "$OS" != "Darwin" ] && [ "$EUID" -ne 0 ]; then
    echo "Proceeding with rootless installation."
else
    # 如果是 root 用户或 macOS，继续检查 macOS 是否需要 root
    if [ "$OS" != "Darwin" ] && [ "$EUID" -ne 0 ]; then
         echo "Error: Please run this script as root (Linux and FreeBSD only)"
         exit 1
    fi
fi

# 映射架构名称到 Go 的命名规范
case "$ARCH" in
    x86_64 | amd64 | x86-64)
        GO_ARCH="amd64"
        ;;
    aarch64 | arm64 | armv8 | armv8l)
        GO_ARCH="arm64"
        ;;
    *)
        echo "Error: Unsupported architecture ($ARCH) on $OS"
        exit 1
        ;;
esac

echo "Downloading nemu version: ${VERSION} ($OS/$GO_ARCH)"

# 获取最新版本号
VERSION_URL="https://raw.githubusercontent.com/WJQSERVER/nemu/main/VERSION"
VERSION=$(curl -s "${VERSION_URL}")

if [ -z "$VERSION" ]; then
    echo "Error: Failed to get latest version from ${VERSION_URL}"
    exit 1
fi

# 下载 VERSION 文件到安装目录
wget -O "${bin_dir}.VERSION" "${VERSION_URL}"

# 构建下载链接
DOWNLOAD_URL="https://github.com/WJQSERVER/nemu/releases/download/${VERSION}/nemu-server-${OS,,}-${GO_ARCH}"

# 下载二进制文件
if ! wget -O "${bin_dir}" "${DOWNLOAD_URL}"; then
    echo "Error: Failed to download nemu from ${DOWNLOAD_URL}"
    exit 1
fi

# 设置执行权限
chmod +x "${bin_dir}"

# 写入systemd unitif [ "$OS" = "Linux" ] && [ "$EUID" -eq 0 ]; then
    systemd_dir="/etc/systemd/system"
    service_file="${systemd_dir}/nemu.service"
    if [ ! -f "$service_file" ]; then
    cat <<EOF >"$service_file"
[Unit]
Description=Nemu Server
After=network.target

[Service]
ExecStart=${bin_dir}
WorkingDirectory=/root/data/nemu
Restart=on-failure
User=root
Group=root

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable nemu.service
    systemctl start nemu.service
else
    echo "Nemu service installed and enabled. Start with: systemctl start nemu.service"
fi

echo "Nemu installed successfully to ${bin_dir}"
echo "Run with: nemu"


