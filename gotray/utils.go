package gotray

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/skratchdot/open-golang/open"
)

// OpenURL 在默认浏览器中打开 URL
func OpenURL(url string) error {
	return open.Run(url)
}

// OpenFile 用默认程序打开文件
func OpenFile(path string) error {
	return open.Run(path)
}

// OpenWithApp 用指定应用打开文件
func OpenWithApp(path, app string) error {
	return open.RunWith(path, app)
}

// SaveEmbedDir 将嵌入的文件系统保存到目标目录
func SaveEmbedDir(efs embed.FS, targetDir string, overwrite bool) error {
	return fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}

		fullPath := filepath.Join(targetDir, path)

		// 检查文件是否存在
		_, statErr := os.Stat(fullPath)
		exists := statErr == nil

		if d.IsDir() {
			if !exists {
				return os.MkdirAll(fullPath, 0755)
			}
			return nil
		}

		// 文件存在且不覆盖
		if exists && !overwrite {
			return nil
		}

		// 读取嵌入文件内容
		content, err := efs.ReadFile(path)
		if err != nil {
			return err
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}

		// 写入文件
		return os.WriteFile(fullPath, content, 0644)
	})
}

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ReadFile 读取文件内容
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile 写入文件
func WriteFile(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}
