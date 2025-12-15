package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// FS 返回 web/dist 目录的文件系统
func FS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// MustFS 返回 web/dist 目录的文件系统，出错时 panic
func MustFS() fs.FS {
	f, err := FS()
	if err != nil {
		panic(err)
	}
	return f
}
