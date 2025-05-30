package decode

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"nemu-server/config"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/WJQSERVER-STUDIO/go-utils/copyb"
	"github.com/WJQSERVER-STUDIO/logger"
	"github.com/infinite-iroha/touka"
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

// MakeDecodeHandler 创建一个标准的 http.HandlerFunc，通过闭包访问配置。
func MakeDecodeHandler(cfg *config.Config) touka.HandlerFunc {
	// 返回符合 http.HandlerFunc 签名的函数
	return func(c *touka.Context) {
		r := c.Request

		// 获取头部信息
		inputToken := r.Header.Get("Nemu-Token")
		if inputToken != cfg.Server.Token {
			logError("Invalid token")
			c.JSON(401, touka.H{
				"message": "Unauthorized",
			})
			return // 提前返回
		}

		// 获取请求体
		reqBody := r.Body
		defer reqBody.Close() // 延迟关闭请求体
		// 标准库的 http server 通常会负责关闭请求体，此处不再手动 defer Close()

		if reqBody == nil {
			logError("Request body is nil")
			c.JSON(http.StatusBadRequest, touka.H{"message": "Request body is nil"})
			return
		}

		// 使用 gzip.NewReader 读取请求体
		gzReader, err := gzip.NewReader(reqBody)
		if err != nil {
			logError("Failed to create gzip reader: %v", err)
			c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to process gzip data: %v", err)})
			return
		}
		defer gzReader.Close() // 延迟关闭 gzip reader

		// 使用 tar.NewReader 读取 tar 数据
		tarReader := tar.NewReader(gzReader)

		// 清理目录
		err = os.RemoveAll(cfg.Server.Dir)
		if err != nil {
			logError("Failed to clean directory: %v", err)
			c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to clean directory: %v", err)})
			return
		}
		err = os.MkdirAll(cfg.Server.Dir, 0755)
		if err != nil {
			logError("Failed to create directory: %v", err)
			c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create directory: %v", err)})
			return
		}

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
				c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to read tar header: %v", err)})
				return
			}

			// 安全路径检查和文件操作
			switch header.Typeflag {
			case tar.TypeReg: // 普通文件
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for file %s: %v", header.Name, err)
					c.JSON(http.StatusBadRequest, touka.H{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}

				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory %s for file %s: %v", targetDir, header.Name, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}

				// 使用 O_TRUNC 标志覆盖现有文件
				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
				if err != nil {
					logError("Failed to create file %s: %v", targetPath, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create file: %v", err)})
					return
				}

				_, err = copyb.Copy(outFile, tarReader)
				outFile.Close() // 在复制完成后立即关闭文件
				if err != nil {
					logError("Failed to copy file %s: %v", targetPath, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to copy file: %v", err)})
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
					c.JSON(http.StatusBadRequest, touka.H{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 使用 MkdirAll 确保父目录也创建
				err = os.MkdirAll(targetPath, os.FileMode(header.Mode))
				if err != nil {
					logError("Failed to create directory %s: %v", targetPath, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create directory: %v", err)})
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
					c.JSON(http.StatusBadRequest, touka.H{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory for symlink %s: %v", targetDir, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}
				// Linkname 是目标路径
				err = os.Symlink(header.Linkname, targetPath)
				if err != nil {
					logError("Failed to create symlink %s -> %s: %v", targetPath, header.Linkname, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create symlink: %v", err)})
					return
				}
				processedEntries++ // 成功处理一个软链接

			case tar.TypeLink: // 硬链接
				targetPath, err := SafeTarExtractPath(cfg.Server.Dir, header.Name)
				if err != nil {
					logError("Path traversal detected for hard link %s: %v", header.Name, err)
					c.JSON(http.StatusBadRequest, touka.H{"message": fmt.Sprintf("Path traversal detected: %v", err)})
					return
				}
				// 确保目标目录存在
				targetDir := filepath.Dir(targetPath)
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					logError("Failed to create directory for hard link %s: %v", targetDir, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create directory: %v", err)})
					return
				}
				// Hard links require the old path relative to the filesystem root or the baseDir.
				// Assuming Linkname is relative to baseDir as original code implies.
				oldPath := filepath.Join(cfg.Server.Dir, header.Linkname)
				err = os.Link(oldPath, targetPath)
				if err != nil {
					logError("Failed to create hard link %s -> %s: %v", targetPath, oldPath, err)
					c.JSON(http.StatusInternalServerError, touka.H{"message": fmt.Sprintf("Failed to create hard link: %v", err)})
					return
				}
				processedEntries++ // 成功处理一个硬链接

			default:
				logWarning("Unhandled tar entry type for %s: %v", header.Name, header.Typeflag)
			}
		}

		// 成功处理所有条目后发送成功响应
		if processedEntries > 0 { // 检查是否处理了至少一个有效条目
			c.JSON(http.StatusOK, touka.H{"message": "success"})
		} else {
			c.JSON(http.StatusBadRequest, touka.H{"message": "No valid entries processed in tar file"})
		}

	}
}
