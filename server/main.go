package main

import (
	"flag"
	"fmt"
	"nemu-server/config"
	"nemu-server/decode"
	"nemu-server/errpage"
	"net/http"

	"os"

	"github.com/WJQSERVER-STUDIO/logger"
	"github.com/fenthope/gzip"
	"github.com/fenthope/record"
	"github.com/infinite-iroha/touka"
)

var (
	cfg     *config.Config
	cfgfile = "config/config.toml"
)

var (
	logw       = logger.Logw
	logDump    = logger.LogDump
	logDebug   = logger.LogDebug
	logInfo    = logger.LogInfo
	logWarning = logger.LogWarning
	logError   = logger.LogError
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

func setupLogger(cfg *config.Config) {
	var err error
	err = logger.Init(cfg.Log.LogFilePath, cfg.Log.MaxLogSize)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	err = logger.SetLogLevel(cfg.Log.Level)
	if err != nil {
		fmt.Printf("Logger Level Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Log Level: %s\n", cfg.Log.Level)
	logDebug("Config File Path: ", cfgfile)
	logDebug("Loaded config: %v\n", cfg)
	logInfo("Logger Initialized Successfully")
}

func init() {
	loadConfig()
	if cfg != nil { // 在setupLogger前添加空值检查
		setupLogger(cfg)
	} else {
		fmt.Println("Config not loaded, exiting.")
		os.Exit(1)
	}
}

func main() {

	r := touka.New()
	r.Use(touka.Recovery())
	r.Use(record.Middleware())

	r.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedExtensions([]string{"css"}),
	))

	//r.Use(touka.Gzip(-1))

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

	defer logger.Close()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	logInfo("Server is running on %s", addr)
	err := r.RunShutdown(addr)
	if err != nil {
		logError("Failed to start server: %v", err)
	} else {
		logInfo("Server stopped")
	}
}
