package entities

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

type ProtoFile struct {
	Path    string // e.g. "user/v1/service.proto"
	Content []byte
}

type ProtoFileSet struct {
	EntryPoint string
	Files      []ProtoFile
}

func (fs ProtoFileSet) ToSourceMap() map[string]string {
	m := make(map[string]string, len(fs.Files))
	for _, f := range fs.Files {
		m[f.Path] = string(f.Content)
	}
	return m
}

func (fs ProtoFileSet) ContentHash() string {
	sorted := make([]ProtoFile, len(fs.Files))
	copy(sorted, fs.Files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Path < sorted[j].Path
	})

	h := sha256.New()
	for _, f := range sorted {
		h.Write([]byte(f.Path))
		h.Write([]byte{0})
		h.Write(f.Content)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}
