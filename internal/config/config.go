package config

import (
	"fmt"
	"github.com/kubackup/kubackup/internal/consts"
	"github.com/kubackup/kubackup/internal/entity/v1/config"
	"github.com/kubackup/kubackup/internal/model"
	"github.com/kubackup/kubackup/pkg/file"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	pathu "path"
	"path/filepath"
)

func ReadConfig(path string) (c *config.Config, err error) {
	path = fileutil.FixPath(path)
	if "" == path {
		c = defaultConfig()
		path = fileutil.FixPath(pathu.Join(fileutil.HomeDir(), string(filepath.Separator), ".kubackup"))
		c.Data.CacheDir = pathu.Join(path, string(filepath.Separator), "cache")
		c.Logger.LogPath = pathu.Join(path, string(filepath.Separator), "log")
		c.Data.DbDir = pathu.Join(path, string(filepath.Separator), "db")
		confpath := pathu.Join(path, string(filepath.Separator), "conf")
		if !fileutil.Exist(confpath) {
			err = os.MkdirAll(confpath, 0777)
			if err != nil {
				return nil, err
			}
		}
		fpath := pathu.Join(confpath, string(filepath.Separator), "app.yml")
		if !fileutil.Exist(fpath) {
			fmt.Printf("加载默认配置: %s\n", path)
			bytes, _ := yaml.Marshal(c)
			err = ioutil.WriteFile(fpath, bytes, 0666)
			if err != nil {
				return nil, err
			}
			return c, nil
		} else {
			c, err = ReadConfig(path)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	} else {
		// 读取path路径配置文件
		v := viper.New()
		v.SetConfigName("app")
		v.SetConfigType("yaml")
		realDir := fileutil.ReplaceHomeDir(pathu.Join(path, string(filepath.Separator), "conf"))
		if exists := fileutil.Exist(realDir); !exists {
			return nil, &model.CustomError{
				Code:    consts.SYSTEM_CODE,
				Message: fmt.Sprintf("读取配置文件%s失败：conf目录不存在，配置时不用写conf", path),
			}
		}
		v.AddConfigPath(realDir)
		if err = v.ReadInConfig(); err != nil {
			fmt.Println(fmt.Errorf("读取配置文件%s失败： %s ,%s", path, realDir, err.Error()))
			return nil, err
		}
		if err = v.Unmarshal(&c); err != nil {
			fmt.Println(fmt.Errorf("读取配置文件%s失败：%s", path, err.Error()))
			return nil, err
		}
		c.Logger.LogPath = pathu.Join(path, string(filepath.Separator), "log")
		c.Data.DbDir = pathu.Join(path, string(filepath.Separator), "db")
		if "" == c.Data.CacheDir {
			c.Data.CacheDir = pathu.Join(path, string(filepath.Separator), "cache")
		}
		fmt.Println(fmt.Sprintf("配置加载完成: %s", path))
	}
	return c, nil
}

// defaultConfig 获取默认配置
func defaultConfig() *config.Config {
	c := &config.Config{}
	c.Server.Name = "kubackup"
	c.Data.NoCache = false
	c.Server.Debug = false
	c.Logger.Level = "info"
	c.Jwt.Key = "dowell"
	c.Jwt.MaxAge = 1800
	c.Server.Bind.Port = 8012
	c.Server.Bind.Host = ""
	c.Prometheus.Enable = false
	return c
}
