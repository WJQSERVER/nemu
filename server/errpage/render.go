package errpage

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"

	"github.com/infinite-iroha/touka"
)

//go:embed page/*
var ErrPageFS embed.FS

// 全局缓存，用于存储按状态码渲染的错误页面
var (
	// errorPageCache 存储状态码到渲染后 HTML []byte 的映射
	errorPageCache = make(map[int][]byte)
	// cacheMutex 用于保护对 errorPageCache 的并发访问
	cacheMutex sync.RWMutex
)

func htmlTemplateRender(fsys fs.FS, data interface{}) ([]byte, error) {
	tmplPath := "page/page.html"
	tmpl, err := template.ParseFS(fsys, tmplPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}
	if tmpl == nil {
		return nil, fmt.Errorf("template is nil")
	}

	// 创建一个 bytes.Buffer 用于存储渲染结果
	var buf bytes.Buffer

	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	// 返回 buffer 的内容作为 []byte
	return buf.Bytes(), nil
}

func ErrorHandler(c *touka.Context, statusCode int) {
	r := c.Request
	w := c.Writer
	// 在写入响应之前检查客户端是否已断开连接
	select {
	case <-r.Context().Done():
		// 如果客户端已断开连接，则不执行任何操作（或仅记录日志）
		// log.Printf("client disconnected before error %d could be sent for %s", statusCode, r.URL.Path)
		return
	default:
	}

	cacheMutex.RLock() // 获取读锁
	cachedPage, found := errorPageCache[statusCode]
	cacheMutex.RUnlock() // 释放读锁

	var htmlContent []byte
	var renderError error

	if found {
		// 缓存命中
		htmlContent = cachedPage
	} else {
		// 缓存未命中，需要渲染页面
		data := FindError(statusCode) // 假设 FindError 返回渲染模板所需的数据

		renderedPage, err := htmlTemplateRender(ErrPageFS, data)
		if err != nil {
			renderError = fmt.Errorf("rendering error page for status %d: %w", statusCode, err)
			// 如果渲染失败，不缓存错误，直接处理这个内部错误
		} else {
			htmlContent = renderedPage
			// 将新渲染的页面存入缓存
			cacheMutex.Lock() // 获取写锁
			errorPageCache[statusCode] = htmlContent
			cacheMutex.Unlock() // 释放写锁
		}
	}

	// 3. 发送响应
	// 首先处理渲染错误（如果有）
	if renderError != nil {
		// 渲染自身失败，返回一个非常简单的文本错误，避免循环依赖 ErrorHandler
		http.Error(w, "An internal error occurred while trying to render the error page.", http.StatusInternalServerError)
		return
	}

	// 如果 htmlContent 仍然是 nil (理论上只有 renderError 非 nil 时才可能，但做个防御)
	if htmlContent == nil {
		http.Error(w, "An unexpected error occurred.", http.StatusInternalServerError)
		return
	}

	/*
		// 检查是否已经存在其他状态码返回
		if w.Written() {
			// 如果响应已经开始写入（WriteHeader 已被调用），则不应该再次写入头部或改变状态码。
			// 这通常发生在中间件或处理函数在调用 ErrorHandler 之前已经写入了部分响应。
			// 在这种情况下，我们只能尝试写入错误信息到已开始的响应体中，但这可能导致客户端解析问题。
			// 更好的做法是，如果响应已开始，直接返回，不尝试写入错误页面。
			logWarning("errpage: response already started for status %d, skipping error page rendering", statusCode)
			return
		}
	*/

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// 确保在 WriteHeader 之前设置所有头部
	// 状态码应该由调用 ErrorHandler 的地方（例如 httprouter）来设置，
	// ErrorHandler 自身不应该再调用 w.WriteHeader(statusCode) 除非它要改变状态码。
	// 但是，通常 ErrorHandler 的职责就是根据传入的 statusCode 生成响应。
	// 这里的 w.WriteHeader(statusCode) 是为了确保响应有正确的状态码。
	// 如果调用者（如 httprouter）已经设置了状态码，重复调用是无害的（net/http会忽略）。
	w.WriteHeader(statusCode)

	_, err := w.Write(htmlContent)
	if err != nil {
		//logWarning("errpage: failed to write response for status %d: %v", statusCode, err)
		c.Warnf("errpage: failed to write response for status %d: %v", statusCode, err)
		return
	}

}
