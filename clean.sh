#!/bin/bash

set -e

if [[ "$OSTYPE" =~ ^linux ]]; then
    systemctl stop kubackup
else
    echo "暂不支持的操作系统，请参阅官方文档：https://kubackup.cn/install/uninstall/"
    exit 1
fi

rm -rf /etc/systemd/system/kubackup.service
rm -rf /usr/local/bin/kubackup_server
rm -rf ~/.kubackup
