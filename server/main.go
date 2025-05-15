package main

import (
	"flag"
	"fmt"
	"nemu-server/config"
	"nemu-server/decode"
	"net/http"

	"os"

	"github.com/WJQSERVER-STUDIO/logger"
	"github.com/WJQSERVER/httprouter"
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

	//r := gin.New()
	//r.Use(gin.Recovery())
	//r.Use(logm.Middleware())

	r := httprouter.New()
	r.HandlerFunc("POST", "/nemu/upload", decode.MakeDecodeHandler(cfg))
	r.HandlerFunc("GET", "/nemu/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	fs := http.Dir(cfg.Server.Dir)
	r.ServeUnmatched(fs)
	/*
		r.POST("/nemu/upload",
			func(c *gin.Context) {
				decode.DecodeHandle(c, cfg)
			})
		r.GET("/nemu/health", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"status": "ok",
			})
		})

		r.Run(cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port))
	*/

	// 自定义protocols
	protocols := new(http.Protocols)
	protocols.SetUnencryptedHTTP2(true)
	protocols.SetHTTP1(true)
	server := &http.Server{
		Addr:      cfg.Server.Host + ":" + fmt.Sprintf("%d", cfg.Server.Port),
		Handler:   r,
		Protocols: protocols,
	}

	go func() {
		logInfo("Server is running on %s %d", cfg.Server.Host, cfg.Server.Port)
	}()
	err := server.ListenAndServe()
	if err != nil {
		logError("Failed to start server: %v", err)
	} else {
		logInfo("Server stopped")
	}

}
