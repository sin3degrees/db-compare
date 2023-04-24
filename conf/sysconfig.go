package conf

import (
	"io/ioutil"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
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
	//指定对应的json配置文件
	b, err := ioutil.ReadFile(dir + "/config.json")
	if err != nil {
		panic("Sys config read err")
	}
	err = jsoniter.Unmarshal(b, Sysconfig)
	if err != nil {
		panic(err)
	}
}

type db struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type sysconfig struct {
	//数据库配置
	DBSrc    db       `json:"db_src"`
	DBDst    db       `json:"db_dst"`
	TbOnly   []string `json:"tb_only"`
	TbIgnore []string `json:"tb_ignore"`
}
