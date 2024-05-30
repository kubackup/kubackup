package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
)

// DownloadFile 文件下载
func DownloadFile(url string, output string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code error: %v", resp.StatusCode)
	}

	out, err := os.Create(output) // 下载后的文件名
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// Get 发送GET请求
func Get(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code error: %v", resp.StatusCode)
	}

	// 读取并打印响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}
	return string(body), nil
}
