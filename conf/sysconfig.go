package conf

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var Sysconfig = &sysconfig{}
var Dir = ""

func init() {
	//获取当前程序的路径
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	Dir = dir
	//指定对应的yml配置文件
	b, err := os.ReadFile(dir + "/config.yml")
	if err != nil {
		panic("Sys config read err")
	}
	err = yaml.Unmarshal(b, Sysconfig)
	if err != nil {
		panic(err)
	}
}

type db struct {
	Type     string `yaml:"type"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type sysconfig struct {
	//数据库配置
	DBSrc    db       `yaml:"db_src"`
	DBDst    db       `yaml:"db_dst"`
	TbOnly   []string `yaml:"tb_only"`
	TbIgnore []string `yaml:"tb_ignore"`
}
