package tui

import "os"

// fileStamp is a cheap change detector for the config file. It is deliberately
// coarse: a change triggers a reload prompt, never data loss.
type fileStamp struct {
	modTime int64
	size    int64
}

func statStamp(path string) fileStamp {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}
	}
	return fileStamp{modTime: info.ModTime().UnixNano(), size: info.Size()}
}
