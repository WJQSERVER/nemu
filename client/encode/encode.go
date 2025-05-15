package encode

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/WJQSERVER-STUDIO/httpc"
	"golang.org/x/crypto/chacha20poly1305"
)

// ClientConfig 客户端配置
type ClientConfig struct {
	ServerURL  string // 服务器地址
	Token      string // 认证 Token
	SourcePath string // 源文件或目录路径
}

// setupHttpClient 创建并配置 HTTP 客户端
func SetupHttpClient() *httpc.Client {
	client := httpc.New()
	return client
}

/*
// sendStreamingTarGz 创建并流式传输 tar.gz 归档
// client: HTTP 客户端
// cfg: 客户端配置
func SendStreamingTarGz(client *httpc.Client, cfg *ClientConfig) error {
	// 创建一个管道
	pr, pw := io.Pipe()

	var (
		resp *http.Response
		err  error
	)

	// 用于接收生产者 goroutine 的错误
	errCh := make(chan error, 1)

	// 启动一个 goroutine 处理打包和压缩
	go func() {
		// 确保在退出时关闭管道写入端
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("producer goroutine panic: %v", r)
				log.Printf("ERROR: Producer goroutine panic: %v", r)
				// 关闭管道并发送错误
				if closeErr := pw.CloseWithError(err); closeErr != nil {
					log.Printf("WARN: Closing pipe writer after panic failed: %v", closeErr)
				}
				// 非阻塞发送错误到通道
				select {
				case errCh <- err:
				default:
				}
			}
		}()

		// 确保按顺序关闭 writer: Tar -> Gzip -> Pipe
		// GzipWriter 写入 pw
		gzWriter := gzip.NewWriter(pw)
		defer func() {
			if err := gzWriter.Close(); err != nil && err != io.ErrClosedPipe {
				log.Printf("ERROR: Closing GzipWriter failed: %v", err)
				select {
				case errCh <- fmt.Errorf("gzip writer close error: %w", err):
				default:
				}
			} else if err == io.ErrClosedPipe {
				log.Println("INFO: Pipe closed while closing GzipWriter")
			}
		}()

		// TarWriter 写入 gzWriter
		tarWriter := tar.NewWriter(gzWriter)
		defer func() {
			if err := tarWriter.Close(); err != nil && err != io.ErrClosedPipe {
				log.Printf("ERROR: Closing TarWriter failed: %v", err)
				select {
				case errCh <- fmt.Errorf("tar writer close error: %w", err):
				default:
				}
			} else if err == io.ErrClosedPipe {
				log.Println("INFO: Pipe closed while closing TarWriter")
			}
		}()

		log.Printf("INFO: Producer starts packing %s", cfg.SourcePath)

		// 遍历源路径
		err := filepath.Walk(cfg.SourcePath, func(file string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("ERROR: Walk error at %s: %v", file, err)
				return fmt.Errorf("walk error: %w", err)
			}

			// 获取文件在归档中的相对路径
			relPath, err := filepath.Rel(cfg.SourcePath, file)
			if err != nil {
				log.Printf("ERROR: Getting relative path for %s failed: %v", file, err)
				return fmt.Errorf("failed to get relative path for %s: %w", file, err)
			}
			// 如果源路径是单个文件，特殊处理相对路径
			if relPath == "." {
				relPath = filepath.Base(cfg.SourcePath)
			}

			// 创建 Tar 头部
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				log.Printf("ERROR: Creating tar header for %s failed: %v", file, err)
				return fmt.Errorf("failed to create tar header: %w", err)
			}
			header.Name = relPath // 设置头部名称为相对路径

			// 处理符号链接
			if info.Mode()&os.ModeSymlink != 0 {
				if header.Typeflag != tar.TypeSymlink {
					log.Printf("WARN: Symlink %s header type is %v, not TypeSymlink", file, header.Typeflag)
				}
				if header.Linkname == "" {
					log.Printf("WARN: Symlink %s header Linkname is empty", file)
				}
				// 符号链接只需写入头部
			} else if info.Mode().IsRegular() {
				// 常规文件需要写入头部和内容
			} else if info.Mode().IsDir() {
				// 目录只需写入头部
			} else {
				// 跳过不支持的文件类型
				log.Printf("WARN: Skipping unsupported file type %v for %s", info.Mode().Type(), file)
				return nil
			}

			// 写入 Tar 头部
			if err := tarWriter.WriteHeader(header); err != nil {
				log.Printf("ERROR: Writing tar header for %s failed: %v", file, err)
				if err == io.ErrClosedPipe { // 管道已关闭，停止写入
					log.Println("INFO: Pipe closed while writing tar header, stopping producer.")
					return filepath.SkipAll
				}
				return fmt.Errorf("failed to write tar header: %w", err)
			}

			// 如果是常规文件，写入文件内容
			if info.Mode().IsRegular() {
				fileHandle, err := os.Open(file)
				if err != nil {
					log.Printf("ERROR: Opening file %s failed: %v", file, err)
					return fmt.Errorf("failed to open file: %w", err)
				}
				defer fileHandle.Close()

				// 复制文件内容到 TarWriter
				if _, err := io.Copy(tarWriter, fileHandle); err != nil {
					log.Printf("ERROR: Copying file content for %s failed: %v", file, err)
					if err == io.ErrClosedPipe { // 管道已关闭，停止复制
						log.Println("INFO: Pipe closed while copying file content, stopping producer.")
						return filepath.SkipAll
					}
					return fmt.Errorf("failed to copy file content: %w", err)
				}
			}

			return nil // 继续遍历
		})

		// 检查 Walk 过程中的错误
		if err != nil {
			log.Printf("ERROR: filepath.Walk finished with error: %v", err)
			// 关闭管道并发送错误
			if closeErr := pw.CloseWithError(err); closeErr != nil {
				log.Printf("WARN: Closing pipe writer after walk error failed: %v", closeErr)
			}
			select { // 非阻塞发送 Walk 错误
			case errCh <- fmt.Errorf("filepath walk error: %w", err):
			default:
			}
			return
		}

		// Walk 成功完成，defer 会关闭 writer 和管道
		log.Println("INFO: Producer finished packing and writing.")
	}() // 生产者 goroutine 结束

	// --- 主 goroutine ---

	rb := client.NewRequestBuilder("POST", cfg.ServerURL)
	rb.SetBody(pr)
	rb.NoDefaultHeaders()
	rb.SetHeader("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 NemuClient/1.0")
	req, err := rb.Build()

	fmt.Println(1)

	// 创建 HTTP 请求，请求体使用管道读取端 pr
	//req, err := http.NewRequest("POST", cfg.ServerURL, pr)
	if err != nil {
		log.Printf("ERROR: Creating HTTP request failed: %v", err)
		// 关闭管道读取端通知生产者停止
		if closeErr := pr.CloseWithError(err); closeErr != nil {
			log.Printf("WARN: Closing pipe reader after request creation failed: %v", closeErr)
		}
		// 检查生产者是否有错误
		select {
		case producerErr := <-errCh:
			return fmt.Errorf("failed to create HTTP request: %w, producer error: %v", err, producerErr)
		default:
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}
	}

	// 设置请求头部
	req.Header.Set("Content-Type", "application/octet-stream")

	// 对token进行sha512处理, 让服务端比对
	nemuTokenHash := sha512.Sum512([]byte(cfg.Token))
	nemuTokenHashStr := fmt.Sprintf("%x", nemuTokenHash)

	req.Header.Set("Nemu-Token", nemuTokenHashStr)

	/*

		// 对token进行sha256处理, 为chacha20准备密钥
		nemuTokenHash256 := sha256.Sum256([]byte(cfg.Token))
		nemuTokenHash256Str := fmt.Sprintf("%x", nemuTokenHash256)

		// 获取当前UTC时间戳
		utcNow := time.Now().UTC()
		// 使用RFC 3339
		utcNowStr := utcNow.Format(time.RFC3339)

		// chacha20加密时间戳
		cipherText, err := SimpleEncrypt([]byte(nemuTokenHash256Str), []byte(utcNowStr))
		if err != nil {
			log.Printf("ERROR: EncryptChaCha20 failed: %v", err)
			return fmt.Errorf("failed to encrypt timestamp: %w", err)
		}
		// 设置请求头部
		req.Header.Set("Nemu-Timestamp", string(cipherText))

*/

//nemuTokenWithTimestamp := SimpleEncrypt([]byte(nemuTokenHash256Str), []byte(fmt.Sprintf(""))
/*

	log.Printf("INFO: Streaming data from %s to %s", cfg.SourcePath, cfg.ServerURL)
	// 读取响应体
	//go func() {
	//time.Sleep(5 * time.Second)

	//}()
	// 发送请求，HTTP 客户端会从 pr 读取数据
	resp, _ = client.Do(req)
	log.Printf("INFO: Server response status: %s", resp.Status)

	fmt.Println(2)
*/

/*
	// 检查发送请求的错误
	if err != nil {
		log.Printf("ERROR: Sending HTTP request failed: %v", err)
		// 检查生产者是否有错误
		select {
		case producerErr := <-errCh:
			return fmt.Errorf("HTTP request failed: %v, producer error: %v", err, producerErr)
		default:
			return fmt.Errorf("failed to send HTTP request: %w", err)
		}
	}
*/
/*

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ERROR: Reading response body failed: %v", err)
		// 检查生产者是否有错误
		select {
		case producerErr := <-errCh:
			return fmt.Errorf("failed to read response body: %v, producer error: %v", err, producerErr)
		default:
			return fmt.Errorf("failed to read response body: %w", err)
		}
	}

	resp.Body.Close()

	log.Printf("INFO: Server response: %s", string(responseBody))

	// 检查服务器状态码
	if resp.StatusCode != http.StatusOK {
		log.Printf("ERROR: Server returned non-OK status: %s, body: %s", resp.Status, string(responseBody))
		// 检查生产者是否有错误
		select {
		case producerErr := <-errCh:
			return fmt.Errorf("server returned non-OK status: %s, body: %s, producer error: %v", resp.Status, string(responseBody), producerErr)
		default:
			return fmt.Errorf("server returned non-OK status: %s, body: %s", resp.Status, string(responseBody))
		}
	}

	log.Printf("Server response: %s", string(responseBody))

	// 最后检查生产者 goroutine 是否报告了错误 (例如，关闭 writer 的错误)
	select {
	case producerErr := <-errCh:
		log.Printf("INFO: Producer reported error after successful response: %v", producerErr)
		return fmt.Errorf("request succeeded, but producer reported post-completion error: %v", producerErr)
	default:
		// 没有错误
		log.Println("INFO: Streaming tar.gz sent successfully, received OK response.")
		return nil
	}
}
*/

// runProducer Goroutine 负责打包、压缩并将数据写入 PipeWriter。
// 它在完成或遇到不可恢复的错误时关闭 PipeWriter。
// 它通过返回 error 来指示其最终状态。
func runProducer(ctx context.Context, pw *io.PipeWriter, cfg *ClientConfig) (producerError error) {
	// 确保在函数退出时（无论是正常还是 panic），都会尝试关闭 PipeWriter
	// 如果已经有关闭错误，则会附加新的错误信息
	defer func() {
		if r := recover(); r != nil {
			producerError = fmt.Errorf("producer panic: %v", r)
			log.Printf("ERROR: Producer panic: %v", r)
		}
		if producerError != nil {
			log.Printf("ERROR: Producer failed with: %v. Closing pipe with error.", producerError)
			if err := pw.CloseWithError(producerError); err != nil {
				log.Printf("WARN: Producer: failed to CloseWithError on pipe writer: %v (original error: %v)", err, producerError)
			}
		} else {
			log.Println("INFO: Producer finished successfully. Closing pipe writer.")
			if err := pw.Close(); err != nil {
				log.Printf("WARN: Producer: failed to Close pipe writer: %v", err)
				// 如果之前没有错误，现在将关闭错误作为生产者错误
				producerError = fmt.Errorf("producer pipe writer close: %w", err)
			}
		}
	}()

	gzWriter := gzip.NewWriter(pw)
	defer func() {
		if err := gzWriter.Close(); err != nil && producerError == nil && !strings.Contains(err.Error(), "pipe closed") {
			// 只有当没有其他错误时，才将此错误设为主要错误
			// 忽略 "pipe closed" 错误，因为它可能是由消费者关闭引起的
			log.Printf("ERROR: Producer: closing GzipWriter failed: %v", err)
			producerError = fmt.Errorf("gzip writer close error: %w", err)
		} else if err != nil {
			log.Printf("INFO: Producer: GzipWriter.Close() error (likely pipe already closed): %v", err)
		}
	}()

	tarWriter := tar.NewWriter(gzWriter)
	defer func() {
		if err := tarWriter.Close(); err != nil && producerError == nil && !strings.Contains(err.Error(), "pipe closed") {
			log.Printf("ERROR: Producer: closing TarWriter failed: %v", err)
			producerError = fmt.Errorf("tar writer close error: %w", err)
		} else if err != nil {
			log.Printf("INFO: Producer: TarWriter.Close() error (likely pipe already closed): %v", err)
		}
	}()

	log.Printf("INFO: Producer: Starting to pack %s", cfg.SourcePath)

	errWalk := filepath.Walk(cfg.SourcePath, func(file string, info os.FileInfo, err error) error {
		// 检查上下文是否已被取消
		select {
		case <-ctx.Done():
			log.Printf("INFO: Producer: Context cancelled during walk at %s. Aborting.", file)
			return ctx.Err() // 中断 Walk
		default:
		}

		if err != nil {
			log.Printf("ERROR: Producer: Walk error accessing %s: %v", file, err)
			return fmt.Errorf("walk access error for %s: %w", file, err)
		}

		relPath, err := filepath.Rel(cfg.SourcePath, file)
		if err != nil {
			log.Printf("ERROR: Producer: Getting relative path for %s failed: %v", file, err)
			return fmt.Errorf("failed to get relative path for %s: %w", file, err)
		}
		if relPath == "." && !info.IsDir() { // 如果源本身是文件
			relPath = filepath.Base(cfg.SourcePath)
		} else if relPath == "." && info.IsDir() {
			// 对于根目录本身，Walk 可能会以 "." 访问它，但我们不希望添加一个名为 "." 的条目
			// 通常，我们会添加其内容，或者如果它是一个空目录，则是一个表示该目录的条目。
			// 如果 SourcePath 是目录，Walk 会首先访问它，然后访问其内容。
			// 如果我们希望 tar 包含一个根文件夹，那么这里的 relPath 需要调整。
			// 简单起见，如果 relPath 是 "." 且是目录，我们跳过，因为它的内容会被单独添加。
			// 或者，如果需要一个顶层文件夹，可以这样做：
			// header.Name = filepath.Base(cfg.SourcePath) + "/" // if it's the root dir itself
			// 但标准的 tar 通常直接放内容，除非指定了父目录。
			// 假设我们直接打包内容。
			if file == cfg.SourcePath && info.IsDir() { // 跳过根目录条目本身，只打包其内容
				return nil
			}
		}

		header, err := tar.FileInfoHeader(info, info.Name()) // 使用 info.Name() 作为 link name (如果它是符号链接)
		if err != nil {
			log.Printf("ERROR: Producer: Creating tar header for %s failed: %v", file, err)
			return fmt.Errorf("failed to create tar header for %s: %w", file, err)
		}
		header.Name = filepath.ToSlash(relPath) // 确保 tar 中的路径是 '/' 分隔的

		// 处理符号链接的目标
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(file)
			if err != nil {
				log.Printf("ERROR: Producer: Reading symlink target for %s failed: %v", file, err)
				return fmt.Errorf("failed to read symlink target for %s: %w", file, err)
			}
			header.Linkname = linkTarget // FileInfoHeader 可能不会正确填充这个，对于某些系统
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			log.Printf("ERROR: Producer: Writing tar header for %s failed: %v", file, err)
			if err == io.ErrClosedPipe || strings.Contains(err.Error(), "pipe closed") {
				log.Println("INFO: Producer: Pipe closed while writing tar header. Aborting walk.")
				return filepath.SkipAll // 使用 SkipAll 而不是直接返回错误，以优雅停止 Walk
			}
			return fmt.Errorf("failed to write tar header for %s: %w", file, err)
		}

		if info.Mode().IsRegular() && info.Size() > 0 { // 仅为非空常规文件复制内容
			fileHandle, err := os.Open(file)
			if err != nil {
				log.Printf("ERROR: Producer: Opening file %s failed: %v", file, err)
				return fmt.Errorf("failed to open file %s: %w", file, err)
			}
			defer fileHandle.Close()

			copiedBytes, err := io.Copy(tarWriter, fileHandle)
			if err != nil {
				log.Printf("ERROR: Producer: Copying content for %s failed: %v (copied %d bytes)", file, err, copiedBytes)
				if err == io.ErrClosedPipe || strings.Contains(err.Error(), "pipe closed") {
					log.Println("INFO: Producer: Pipe closed while copying file content. Aborting walk.")
					return filepath.SkipAll
				}
				return fmt.Errorf("failed to copy content for %s: %w", file, err)
			}
			// log.Printf("DEBUG: Producer: Copied %d bytes for file %s", copiedBytes, file)
		}
		return nil
	})

	if errWalk != nil {
		// 如果 errWalk 是 filepath.SkipAll 或 context.Canceled/DeadlineExceeded，这不是一个“意外”错误
		if errWalk == filepath.SkipAll || errWalk == context.Canceled || errWalk == context.DeadlineExceeded {
			log.Printf("INFO: Producer: Filepath.Walk aborted as expected: %v", errWalk)
			// producerError 将在 defer 中通过 pw.Close() (如果 pipe 已被消费者关闭) 或 pw.CloseWithError(ctx.Err()) 设置
			if errWalk == context.Canceled || errWalk == context.DeadlineExceeded {
				producerError = errWalk // 将上下文错误作为主要错误
			}
			// 如果是 SkipAll，通常意味着 pipe 已经关闭，让 defer 处理 pw.Close()
		} else {
			log.Printf("ERROR: Producer: filepath.Walk completed with error: %v", errWalk)
			producerError = fmt.Errorf("filepath.Walk error: %w", errWalk) // 设置主要错误
		}
	}

	// 如果到这里 producerError 仍然是 nil，那么打包过程（Walk 和写入Header/Content）是成功的。
	// 接下来 defer 中的 tarWriter.Close(), gzWriter.Close(), pw.Close() 将会执行。
	// 如果这些 Close 操作失败，它们可能会设置 producerError。

	if producerError == nil {
		log.Println("INFO: Producer: Packing and writing completed successfully (prior to final closes).")
	}
	return producerError // 返回在 walk 或 panic 期间发生的任何错误
}

// SendStreamingTarGz 创建并流式传输 tar.gz 归档
func SendStreamingTarGz(parentCtx context.Context, httpClient *httpc.Client, cfg *ClientConfig) error {
	// 使用 context 控制生产者 goroutine 的生命周期
	ctx, cancelProducer := context.WithCancel(parentCtx)
	defer cancelProducer() // 确保在函数退出时，生产者 goroutine 会被通知取消

	pr, pw := io.Pipe()

	producerErrCh := make(chan error, 1) // 用于接收生产者 goroutine 的最终错误

	// 启动生产者 goroutine
	go func() {
		// runProducer 会处理自己的 panic 并将错误返回
		// 它也会确保 pw 被关闭 (Close or CloseWithError)
		err := runProducer(ctx, pw, cfg)
		if err != nil {
			log.Printf("INFO: Producer goroutine finished with error: %v", err)
		} else {
			log.Printf("INFO: Producer goroutine finished successfully.")
		}
		producerErrCh <- err
		close(producerErrCh)
	}()

	// --- 主 goroutine (消费者) ---
	log.Printf("INFO: Main: Preparing to stream data from %s to %s", cfg.SourcePath, cfg.ServerURL)

	rb := httpClient.NewRequestBuilder("POST", cfg.ServerURL)
	rb.SetBody(pr)        // pr 会从生产者 goroutine 写入的 pw 读取数据
	rb.NoDefaultHeaders() // 假设这是必要的
	rb.SetHeader("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36 NemuClient/1.0")
	rb.SetHeader("Content-Type", "application/octet-stream") // 通常流式传输使用这个

	nemuTokenHash := sha512.Sum512([]byte(cfg.Token))
	rb.SetHeader("Nemu-Token", fmt.Sprintf("%x", nemuTokenHash))
	// 其他头部设置 (如 Nemu-Timestamp, 如果需要) 可以加在这里

	req, err := rb.Build()
	if err != nil {
		log.Printf("ERROR: Main: Creating HTTP request failed: %v", err)
		// 请求创建失败，通知生产者停止并关闭管道读取端
		// cancelProducer() 已经在 defer 中，这里显式调用可以更快地停止
		cancelProducer()
		// 关闭 pr 端也会让 pw 端的写入失败，从而停止生产者
		_ = pr.CloseWithError(fmt.Errorf("HTTP request creation failed: %w", err)) // 忽略 pr.CloseWithError 的错误，因为我们关心的是原始错误

		// 等待生产者完成（它可能会因为管道关闭或上下文取消而错误退出）
		producerErr := <-producerErrCh
		if producerErr != nil && producerErr != context.Canceled && !strings.Contains(producerErr.Error(), "pipe closed") {
			return fmt.Errorf("failed to create HTTP request: %v, and producer also failed: %v", err, producerErr)
		}
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req = req.WithContext(ctx) // 将上下文传递给 HTTP 请求，允许取消请求

	// 发送请求
	log.Printf("INFO: Main: Sending HTTP POST request to %s", cfg.ServerURL)
	resp, httpErr := httpClient.Do(req)

	// 在处理 HTTP 响应或错误之后，我们需要确保管道读取端被关闭，
	// 以便生产者（如果仍在运行）不会永远阻塞在写入端。
	// 通常，如果 http.Client.Do 成功读取了所有请求体（来自 pr），
	// 并且生产者关闭了 pw，那么 pr 也会到达 EOF。
	// 如果 Do 返回错误，它可能已经关闭了 pr，或者没有。
	// 调用 pr.Close() 或 pr.CloseWithError() 是安全的，可以确保。
	// cancelProducer() 也会触发生产者的清理。

	if httpErr != nil {
		log.Printf("ERROR: Main: HTTP request failed: %v", httpErr)
		// 请求失败，通知生产者取消（如果它还在运行）
		cancelProducer() // 确保生产者被信号取消
		// 关闭 pr 以便生产者写入时出错并退出
		_ = pr.CloseWithError(fmt.Errorf("HTTP request execution failed: %w", httpErr))

		// 等待生产者完成并获取其错误
		producerErr := <-producerErrCh // 会阻塞直到 producer goroutine 发送错误并退出
		if producerErr != nil && producerErr != context.Canceled && !strings.Contains(producerErr.Error(), "pipe closed") {
			// 如果生产者错误不是因为取消或管道关闭，则包含它
			return fmt.Errorf("HTTP request failed: %v, and producer also failed: %v", httpErr, producerErr)
		}
		return fmt.Errorf("HTTP request failed: %w", httpErr)
	}
	defer resp.Body.Close()

	log.Printf("INFO: Main: Received HTTP response: Status %s", resp.Status)

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("ERROR: Main: Reading response body failed: %v", readErr)
		// 即使读取响应体失败，也要检查生产者错误
		cancelProducer() // 确保生产者停止
		_ = pr.Close()   // 确保管道读取端关闭

		producerErr := <-producerErrCh
		if producerErr != nil && producerErr != context.Canceled && !strings.Contains(producerErr.Error(), "pipe closed") {
			return fmt.Errorf("failed to read response body: %v, and producer also failed: %v", readErr, producerErr)
		}
		return fmt.Errorf("failed to read response body: %w", readErr)
	}

	log.Printf("INFO: Main: Server response body: %s", string(responseBody))

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf("server returned non-OK status: %s, body: %s", resp.Status, string(responseBody))
		log.Printf("ERROR: Main: %s", errMsg.Error())
		cancelProducer()
		_ = pr.Close()

		producerErr := <-producerErrCh
		if producerErr != nil && producerErr != context.Canceled && !strings.Contains(producerErr.Error(), "pipe closed") {
			return fmt.Errorf("%w (producer error: %v)", errMsg, producerErr)
		}
		return errMsg
	}

	// 等待生产者 goroutine 完成并检查其最终错误
	// 即使 HTTP 请求成功，生产者也可能在关闭其 writer 时遇到问题
	producerErr := <-producerErrCh
	if producerErr != nil {
		// 只有当错误不是由于上下文取消或管道已关闭（这可能是由HTTP客户端成功读取所有数据后，我们关闭管道引起的）
		if producerErr == context.Canceled || strings.Contains(producerErr.Error(), "pipe closed") {
			log.Printf("INFO: Main: Producer finished with expected error after successful HTTP request: %v", producerErr)
		} else {
			log.Printf("ERROR: Main: HTTP request succeeded, but producer reported an error: %v", producerErr)
			return fmt.Errorf("HTTP request successful, but producer failed: %w", producerErr)
		}
	}

	log.Println("INFO: Main: Streaming tar.gz sent successfully, and received OK response.")
	return nil
}

// SimpleEncrypt 使用 ChaCha20-Poly1305 加密数据
// 返回值是 []byte，包含 Nonce、密文和认证标签
func SimpleEncrypt(key, plaintext []byte) ([]byte, error) {
	// 密钥长度必须是 32 字节
	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("无效的密钥长度：%d，需要 %d 字节", len(key), chacha20poly1305.KeySize)
	}

	// 创建 AEAD 对象
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("创建 ChaCha20-Poly1305 失败: %w", err)
	}

	// 生成 Nonce (一次性随机数)
	// Nonce 长度必须是 AEAD 规定的长度
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成 Nonce 失败: %w", err)
	}

	// 加密：将 Nonce 放在密文前面，方便接收方提取
	// Seal 方法会将密文和认证标签追加到第一个参数（这里是 nonce）后面
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil) // 最后一个 nil 是可选的附加数据

	return ciphertext, nil
}
