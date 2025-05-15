package main

import (
	"context"
	"crypto/sha512"
	"flag"
	"fmt"
	"nemu-client/encode"
	"nemu-client/render"
	"os"
	"strings"
)

var (
	password string
	host     string
	help     bool
	hash     bool
	debug    bool
	delete   bool
	norender bool
)

func parseFlag() {
	// --password / -p 密码
	// --host / -h 目标域名
	// --help 帮助信息
	// --hash 把输入的密码转换为sha512
	// --debug 允许跳过host检查
	// --delete / -d 删除public目录
	// --norender 不进行渲染

	flag.StringVar(&password, "password", "", "密码")
	flag.StringVar(&password, "p", "", "密码")
	flag.StringVar(&host, "host", "", "目标域名")
	flag.StringVar(&host, "h", "", "目标域名")
	flag.BoolVar(&help, "help", false, "帮助信息")
	flag.BoolVar(&hash, "hash", false, "把输入的密码转换为sha512")
	flag.BoolVar(&debug, "debug", false, "允许跳过host检查")
	flag.BoolVar(&delete, "delete", false, "删除public目录")
	flag.BoolVar(&norender, "norender", false, "不进行渲染")

	flag.Parse()

	if help {
		flag.Usage()
		return
	}
}

func main() {

	parseFlag()

	// 生成hash
	if hash {
		if password != "" {

			nemuTokenHash := sha512.Sum512([]byte(password))
			nemuTokenHashStr := fmt.Sprintf("%x", nemuTokenHash)
			fmt.Println(nemuTokenHashStr)
			os.Exit(0)

		} else {
			fmt.Println("密码不能为空")
			os.Exit(1)
		}
	}

	// 处理host (example.com https://example.com http://example.com 转换为 https://example.com/nemu/upload)
	if host != "" && !debug {
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "https://" + host
		}
		if !strings.HasSuffix(host, "/nemu/upload") {
			host += "/nemu/upload"
		}
	} else if host != "" && debug {
		host = host + "/nemu/upload"

	} else {
		flag.Usage()
		return
	}

	// 处理密码
	if password == "" {
		println("密码不能为空")
		return
	}

	// 处理当前目录
	dir, err := os.Getwd()
	pubdir := dir + "/public"
	if err != nil {
		println("获取当前目录失败")
		return
	}

	if !norender {
		if err := render.HugoRender(dir); err != nil {
			println("渲染失败")
			println(err.Error())
			return
		}
	}

	// 检测目录是否存在
	if _, err := os.Stat(pubdir); os.IsNotExist(err) {
		println("public目录不存在")
		return
	}

	// 构造客户端配置
	cfg := encode.ClientConfig{
		ServerURL:  host,
		Token:      password,
		SourcePath: pubdir,
	}

	// 创建 HTTP 客户端
	client := encode.SetupHttpClient()

	// 发送数据
	err = encode.SendStreamingTarGz(context.Background(), client, &cfg)
	if err != nil {
		println("发送数据失败")
		println(err.Error())
		return
	}

	fmt.Printf("发送数据成功\n")

	// 删除public目录
	if delete {
		err = os.RemoveAll(pubdir)
		if err != nil {
			println("删除public目录失败")
			println(err.Error())
			return
		}
	}

	os.Exit(0)

}
