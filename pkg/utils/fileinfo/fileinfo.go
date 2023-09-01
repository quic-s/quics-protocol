package fileinfo

import (
	"bytes"
	"encoding/gob"
	"log"
	"os"
	"time"
)

type FileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// Encode your own type using gob
func EncodeFileInfo(src os.FileInfo) ([]byte, error) {
	fileInfo := &FileInfo{
		Name:    src.Name(),
		Size:    src.Size(),
		Mode:    src.Mode(),
		ModTime: src.ModTime(),
		IsDir:   src.IsDir(),
	}

	gobBuf := new(bytes.Buffer)
	enc := gob.NewEncoder(gobBuf)
	err := enc.Encode(fileInfo)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}

	return gobBuf.Bytes(), nil
}

func DecodeFileInfo(src []byte) (*FileInfo, error) {
	dec := gob.NewDecoder(bytes.NewBuffer(src))

	fileInfo := &FileInfo{}
	err := dec.Decode(fileInfo)
	if err != nil {
		log.Println("quics-protocol: ", err)
		return nil, err
	}

	return fileInfo, nil
}
