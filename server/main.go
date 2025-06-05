package main

import (
	"flag"
	"fmt"
	"nemu-server/config"
	"nemu-server/decode"
	"nemu-server/errpage"
	"net/http"
	"strings"
	"time"

	"os"

	"github.com/fenthope/compress"
	"github.com/fenthope/reco"
	"github.com/fenthope/record"
	"github.com/infinite-iroha/touka"
)

var (
	cfg     *config.Config
	cfgfile = "config/config.toml"
)

func loadConfig() {
	var err error
	cfg, err = config.LoadConfig(cfgfile)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		// 如果配置文件加载失败，也显示帮助信息并退出
		flag.Usage()
		os.Exit(1)
	}
}

func init() {
	loadConfig()
}

// Cache-Control中间件, 针对特定文件扩展名特殊处理
func CacheControlMiddleware(maxAge int, extensions ...string) touka.HandlerFunc {
	return func(c *touka.Context) {
		// 判断路径文件扩展名
		path := c.Request.URL.Path
		applyCache := false
		for _, ext := range extensions {
			if strings.HasSuffix(path, "."+ext) {
				applyCache = true
				break
			}
		}

		if applyCache {
			c.SetHeader("Cache-Control", fmt.Sprintf("public, max-age=%d, must-revalidate", maxAge))
		} else {
			c.SetHeader("Cache-Control", "public, max-age=3600, must-revalidate")
		}

		c.Next()
	}
}

func main() {

	r := touka.New()
	r.Use(touka.Recovery())
	r.Use(record.Middleware())
	r.Use(CacheControlMiddleware(36000, "woff", "woff2", "ttf", "eot", "otf"))
	r.Use(compress.Compression(
		compress.DefaultCompressionConfig(),
	))

	r.SetLogger(reco.Config{
		Level:           reco.LevelInfo,
		Mode:            reco.ModeText,
		TimeFormat:      time.RFC3339,
		FilePath:        "nemu.log",
		EnableRotation:  true,
		MaxFileSizeMB:   5,
		MaxBackups:      5,
		CompressBackups: true,
		Async:           true,
		DefaultFields:   nil,
	})

	r.POST("/nemu/upload", decode.MakeDecodeHandler(cfg))
	r.GET("/nemu/health", func(c *touka.Context) {
		c.String(http.StatusOK, "ok")
	})

	fs := http.Dir(cfg.Server.Dir)
	r.SetUnMatchFS(fs)
	r.SetErrorHandler(errpage.ErrorHandler)
	r.SetProtocols(&touka.ProtocolsConfig{
		Http1:           true,
		Http2_Cleartext: true,
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	r.LogReco.Infof("Server is running on %s", addr)
	err := r.RunShutdown(addr)
	if err != nil {
		r.LogReco.Errorf("Failed to start server: %v", err)
	} else {
		r.LogReco.Infof("Server stopped")
	}
}
