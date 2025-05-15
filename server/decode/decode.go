package decode

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"nemu-server/config"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/WJQSERVER-STUDIO/go-utils/copyb"
	"github.com/WJQSERVER-STUDIO/logger"
)

var (
	logw       = logger.Logw
	logDump    = logger.LogDump
	logDebug   = logger.LogDebug
	logInfo    = logger.LogInfo
	logWarning = logger.LogWarning
	logError   = logger.LogError
)

func SafeTarExtractPath(baseDir string, tarEntryName string) (string, error) {
	// 获取基础路径的绝对路径并进行清理
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for base directory '%s': %w", baseDir, err)
	}
	cleanBasePath := filepath.Clean(absBase)

	// 清理归档条目名称
	cleanEntryName := filepath.Clean(tarEntryName)

	// 将清理后的基础路径和清理后的条目名称拼接
	joinedPath := filepath.Join(cleanBasePath, cleanEntryName)

	// 再次清理拼接后的路径，这是防止路径遍历的关键步骤
	finalPath := filepath.Clean(joinedPath)

	// 验证最终路径是否确实位于清理后的基础路径之下
	// 使用 filepath.Rel 检查相对路径。
	rel, err := filepath.Rel(cleanBasePath, finalPath)
	if err != nil {
		return "", fmt.Errorf("failed to calculate relative path for '%s' from base '%s': %w", tarEntryName, baseDir, err)
	}

	// 如果相对路径以 ".." 开头，则说明发生了路径遍历
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal detected: '%s' resolves outside base directory '%s'", tarEntryName, baseDir)
	}

	// 如果所有检查都通过，则 finalPath 是安全的，返回该路径
	return finalPath, nil
}

/*

func DecodeHandle(c *gin.Context, cfg *config.Config) {

	inputToken := c.GetHeader("Nemu-Token")
	if inputToken != cfg.Server.Token {
		logError("Invalid token")
		c.JSON(401, gin.H{
			"message": "Unauthorized",
		})
		return
	}

	reqBody := c.Request.Body
	defer reqBody.Close()

	if reqBody == nil {
		logError("Request body is nil")
		return
	}

	gzReader, err := gzip.NewReader(reqBody)
	if err != nil {
		logError("Failed to create gzip reader: %v", err)
		return
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	// 解压到 cfg.Server.Dir 文件夹内
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			logError("Failed to read tar header: %v", err)
			return
		}
		switch header.Typeflag {
		case tar.TypeReg:

			targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
			if err != nil {
				logError("Path traversal detected: %v", err)
				return
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				logError("Failed to create file: %v", err)
				return
			}

			defer outFile.Close()

			_, err = copyb.Copy(outFile, tarReader)
			if err != nil {
				logError("Failed to copy file: %v", err)
				return
			}

			err = os.Chmod(targetPath, os.FileMode(header.Mode))
			if err != nil {
				logError("Failed to change file permissions: %v", err)
				return
			}

			if !header.ModTime.IsZero() {
				err := os.Chtimes(targetPath, header.ModTime, header.ModTime)
				if err != nil {
					logError("Failed to change file modification time: %v", err)
					return
				}
			}

		case tar.TypeDir:
			targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
			if err != nil {
				logError("Path traversal detected: %v", err)
				return
			}
			err = os.MkdirAll(targetPath, os.FileMode(header.Mode))
			if err != nil {
				logError("Failed to create directory: %v", err)
				return
			}

			err = os.Chmod(targetPath, os.FileMode(header.Mode))
			if err != nil {
				logError("Failed to change directory permissions: %v", err)
				return
			}

			if !header.ModTime.IsZero() {
				err = os.Chtimes(targetPath, header.ModTime, header.ModTime)
				if err != nil {
					logError("Failed to change directory modification time: %v", err)
					return
				}

			}

		case tar.TypeSymlink:
			targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
			if err != nil {
				logError("Path traversal detected: %v", err)
				return
			}
			err = os.Symlink(header.Linkname, targetPath)
			if err != nil {
				logError("Failed to create symlink: %v", err)
				return
			}

		case tar.TypeLink:
			targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
			if err != nil {
				logError("Path traversal detected: %v", err)
				return
			}
			err = os.Link(cfg.Server.Dir+"/"+header.Linkname, targetPath)
			if err != nil {
				logError("Failed to create hard link: %v", err)
				return
			}

		default:
			logWarning("Unhandled type: %v", header.Typeflag)

		}

		// 返回状态码
		c.JSON(200, gin.H{
			"message": "success",
		})

	}

}
*/

// MakeDecodeHandler 创建一个标准的 http.HandlerFunc，通过闭包访问配置。
func MakeDecodeHandler(cfg *config.Config) http.HandlerFunc {
	// 返回符合 http.HandlerFunc 签名的函数
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取头部信息
		inputToken := r.Header.Get("Nemu-Token")
		if inputToken != cfg.Server.Token {
			logError("Invalid token")
			// 发送 JSON 响应 (手动设置头部和状态码，编码 JSON)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized) // 401
			json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized"})
			return // 提前返回
		}

		// 获取请求体
		reqBody := r.Body
		defer reqBody.Close() // 延迟关闭请求体
		// 标准库的 http server 通常会负责关闭请求体，此处不再手动 defer Close()

		if reqBody == nil {
			logError("Request body is nil")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) // 400
			json.NewEncoder(w).Encode(map[string]string{"message": "Request body is nil"})
			return
		}

		// 使用 gzip.NewReader 读取请求体
		gzReader, err := gzip.NewReader(reqBody)
		if err != nil {
			logError("Failed to create gzip reader: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError) // 500
			json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to process gzip data: %v", err)})
			return
		}
		defer gzReader.Close() // 延迟关闭 gzip reader

		// 使用 tar.NewReader 读取 tar 数据
		tarReader := tar.NewReader(gzReader)

		// 解压到 cfg.Server.Dir 文件夹内
		processedEntries := 0 // 跟踪是否成功处理了至少一个条目
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				logInfo("All entries processed successfully")
				break // 文件结束
			}
			if err != nil {
				logError("Failed to read tar header: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError) // 500
				json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to read tar header: %v", err)})
				return
			}

			// 安全路径检查和文件操作
			switch header.Typeflag {
			case tar.TypeReg: // 普通文件
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for file %s: %v", header.Name, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest) // 400
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}

				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory %s for file %s: %v", targetDir, header.Name, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}

				// 使用 O_TRUNC 标志覆盖现有文件
				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					logError("Failed to create file %s: %v", targetPath, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create file: %v", err)})
					return
				}

				_, err = copyb.Copy(outFile, tarReader)
				outFile.Close() // 在复制完成后立即关闭文件
				if err != nil {
					logError("Failed to copy file %s: %v", targetPath, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to copy file: %v", err)})
					return
				}

				// 权限和时间戳
				err = os.Chmod(targetPath, os.FileMode(header.Mode))
				if err != nil {
					logWarning("Failed to change file permissions for %s: %v", targetPath, err)
				}
				if !header.ModTime.IsZero() {
					err := os.Chtimes(targetPath, header.ModTime, header.ModTime)
					if err != nil {
						logWarning("Failed to change file modification time for %s: %v", targetPath, err)
					}
				}
				processedEntries++ // 成功处理一个文件

			case tar.TypeDir: // 目录
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for directory %s: %v", header.Name, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest) // 400
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 使用 MkdirAll 确保父目录也创建
				err = os.MkdirAll(targetPath, os.FileMode(header.Mode))
				if err != nil {
					logError("Failed to create directory %s: %v", targetPath, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}

				// 权限和时间戳
				err = os.Chmod(targetPath, os.FileMode(header.Mode))
				if err != nil {
					logWarning("Failed to change directory permissions for %s: %v", targetPath, err)
				}
				if !header.ModTime.IsZero() {
					err = os.Chtimes(targetPath, header.ModTime, header.ModTime)
					if err != nil {
						logWarning("Failed to change directory modification time for %s: %v", targetPath, err)
					}
				}
				processedEntries++ // 成功处理一个目录

			case tar.TypeSymlink: // 软链接
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for symlink %s: %v", header.Name, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest) // 400
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory for symlink %s: %v", targetDir, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}
				// Linkname 是目标路径
				err = os.Symlink(header.Linkname, targetPath)
				if err != nil {
					logError("Failed to create symlink %s -> %s: %v", targetPath, header.Linkname, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create symlink: %v", err)})
					return
				}
				processedEntries++ // 成功处理一个软链接

			case tar.TypeLink: // 硬链接
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for hard link %s: %v", header.Name, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusBadRequest) // 400
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory for hard link %s: %v", targetDir, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}
				// Hard links require the old path relative to the filesystem root or the baseDir.
				// Assuming Linkname is relative to baseDir as original code implies.
				oldPath := filepath.Join(cfg.Server.Dir, header.Linkname)
				err = os.Link(oldPath, targetPath)
				if err != nil {
					logError("Failed to create hard link %s -> %s: %v", targetPath, oldPath, err)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError) // 500
					json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Failed to create hard link: %v", err)})
					return
				}
				processedEntries++ // 成功处理一个硬链接

			default:
				logWarning("Unhandled tar entry type for %s: %v", header.Name, header.Typeflag)
				// 对于未处理类型，可以选择记录警告并继续，或者返回错误
				// w.Header().Set("Content-Type", "application/json")
				// w.WriteHeader(http.StatusBadRequest) // 400
				// json.NewEncoder(w).Encode(map[string]string{"message": fmt.Sprintf("Unsupported tar entry type for %s", header.Name)})
				// return // 如果选择返回错误
			}
		}

		// 成功处理所有条目后发送成功响应
		if processedEntries > 0 { // 检查是否处理了至少一个有效条目
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // 200
			json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest) // 400 如果没有处理任何有效条目，可能表示请求有问题
			json.NewEncoder(w).Encode(map[string]string{"message": "No valid entries processed in tar file"})
		}

	}
}
