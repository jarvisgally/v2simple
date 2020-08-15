package common

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// Function that search the specified file in related paths.
func GetPath(fileName string) string {
	paths := make([]string, 0)
	// Same folder with exec file
	if execFile, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(execFile), fileName))
	}
	// Same folder of the source file
	if _, srcFile, _, ok := runtime.Caller(0); ok {
		paths = append(paths, filepath.Join(filepath.Dir(srcFile), fileName))
	}
	// Same folder of working folder
	if workingDir, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(workingDir, fileName))
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			log.Printf("using %v\n", p)
			return p
		}
	}
	return ""
}
