#!/bin/bash

set -e

osCheck=`uname -m`
if [[ $osCheck =~ 'x86_64' ]];then
    architecture="amd64"
elif [[ $osCheck == 'aarch64' ]];then
    architecture="arm64"
else
    echo "暂不支持的系统架构，请参阅官方文档：https://kubackup.cn/installation/online/"
    exit 1
fi

if [[ "$OSTYPE" =~ ^linux ]]; then
    os="linux"
elif [[ "$OSTYPE" =~ ^darwin ]]; then
    os="darwin"
else
    echo "暂不支持的操作系统，请参阅官方文档：https://kubackup.cn/installation/online/"
    exit 1
fi

VERSION=$(curl -s https://cos.kubackup.cn/script/latest)
if [[ "y${VERSION}" == "y" ]];then
    echo "获取最新版本失败，请稍候重试"
    exit 1
fi

function install(){
    mv $1 /usr/local/bin/kubackup_server && chmod +x /usr/local/bin/kubackup_server
    if [[ "$OSTYPE" =~ ^linux ]]; then
        mv kubackup.service /etc/systemd/system/
        systemctl enable kubackup
        systemctl daemon-reload
        systemctl start kubackup
        for b in {1..30}
        do
            sleep 3
            service_status=`systemctl status kubackup 2>&1 | grep Active`
            if [[ $service_status == *running* ]];then
                echo "kubackup 服务启动成功!"
                systemctl status kubackup
                break;
            else
                echo "kubackup 服务启动出错!"
                exit 1
            fi
        done
    elif [[ "$OSTYPE" =~ ^darwin ]]; then
        echo "命令行下执行：kubackup_server，运行服务"
        kubackup_server
    else
        echo "暂不支持的操作系统，请参阅官方文档：https://kubackup.cn/installation/online/"
        exit 1
    fi
}

package_file_name="kubackup_server_${VERSION}_${os}_${architecture}"
HASH_FILE_URL="https://gitee.com/kubackup/kubackup/releases/download/${VERSION}/${package_file_name}.sum"
package_download_url="https://gitee.com/kubackup/kubackup/releases/download/${VERSION}/${package_file_name}"
service_file_url="https://cos.kubackup.cn/script/kubackup.service"
expected_hash=$(curl -s "$HASH_FILE_URL" | awk '{print $1}')

if [ -f ${package_file_name} ];then
    actual_hash=$(sha256sum "$package_file_name" | awk '{print $1}')
    if [[ "$expected_hash" == "$actual_hash" ]];then
        echo "安装包已存在，跳过下载"
        install ${package_file_name}
        exit 0
    else
        echo "已存在安装包，但是哈希值不一致，开始重新下载"
        rm -f ${package_file_name}
    fi
fi

echo "开始下载 kubackup ${VERSION} 版本安装包"
echo "安装包下载地址： ${package_download_url}"

curl -Lk -o ${package_file_name} ${package_download_url}

if [[ "$OSTYPE" =~ ^linux ]]; then
    curl -Lk -o kubackup.service ${service_file_url}
fi

if [ ! -f ${package_file_name} ];then
	echo "下载安装包失败，请稍候重试。"
	exit 1
fi

install ${package_file_name}
