package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/zeebo/blake3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func hashFile(path, prefix string) ([]byte, error) {
	h := blake3.New()
	r, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	hf := blake3.New()
	_, err = io.Copy(hf, r)
	r.Close()
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(h, "f: %x %s\n", hf.Sum(nil), strings.TrimPrefix(path, prefix))
	return h.Sum(nil), nil
}

type HashFunc func(files []string, prefix string, open func(string) (io.ReadCloser, error)) ([]byte, error)

func hashDir(dir, prefix string) ([]byte, error) {
	h := blake3.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		hf := sha256.New()
		_, err = io.Copy(hf, r)
		r.Close()
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%x  %s\n", hf.Sum(nil), strings.TrimPrefix(path, prefix))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func hashDirectory(path, prefix string) ([]byte, error) {
	hash, err := hashDir(path, prefix)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func DirHash(path, prefix string) (mtime TimeStamp, notExist bool, err error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return 0, true, nil // 对应于 C++ 中的 ERROR_FILE_NOT_FOUND 或 ERROR_PATH_NOT_FOUND
		}
		return -1, true, err
	}
	h2 := fnv1a.Init64
	if info.IsDir() {
		hash, err := hashDirectory(path, prefix)
		if err != nil {
			return -1, true, err
		}
		h2 = fnv1a.AddBytes64(h2, hash)
	} else {
		hash, err := hashFile(path, prefix)
		if err != nil {
			return -1, true, err
		}
		h2 = fnv1a.AddBytes64(h2, hash)
	}
	return TimeStamp(h2), false, nil
}

func NodesHash(nodes []*Node, prefix string) (mtime TimeStamp, notExist bool, err error) {
	h2 := fnv1a.Init64
	for _, node := range nodes {
		hash, err := hashFile(node.path(), prefix)
		if err != nil {
			return -1, true, err
		}
		h2 = fnv1a.AddBytes64(h2, hash)
	}
	return TimeStamp(h2), false, nil
}
