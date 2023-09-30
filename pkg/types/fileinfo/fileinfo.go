package fileinfo

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	pb "github.com/quic-s/quics-protocol/proto/v1"
)

type FileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// Encode your own type using gob
func NewFromOSFileInfo(src os.FileInfo) (*FileInfo, error) {
	fileInfo := &FileInfo{
		Name:    src.Name(),
		Size:    src.Size(),
		Mode:    src.Mode(),
		ModTime: src.ModTime(),
		IsDir:   src.IsDir(),
	}
	return fileInfo, nil
}

func NewFromProtobuf(src *pb.FileInfo) (*FileInfo, error) {
	modTime := time.Time{}
	err := modTime.GobDecode(src.ModTime)
	if err != nil {
		return nil, err
	}

	fileInfo := &FileInfo{
		Name:    src.Name,
		Size:    src.Size,
		Mode:    os.FileMode(src.Mode),
		ModTime: modTime,
		IsDir:   src.IsDir,
	}

	return fileInfo, nil
}

func (f *FileInfo) ToProtobuf() (*pb.FileInfo, error) {
	gobModTime, err := f.ModTime.GobEncode()
	if err != nil {
		return nil, err
	}

	fileInfo := &pb.FileInfo{
		Name:    f.Name,
		Size:    f.Size,
		Mode:    int32(f.Mode),
		ModTime: gobModTime,
		IsDir:   f.IsDir,
	}
	return fileInfo, nil
}

func (f *FileInfo) WriteFileWithInfo(filePath string, fileContent io.Reader) error {
	if f.IsDir {
		err := os.MkdirAll(filePath, f.Mode)
		if err != nil {
			return err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		err = file.Chmod(f.Mode)
		if err != nil {
			return err
		}
		err = os.Chtimes(filePath, time.Now(), f.ModTime)
		if err != nil {
			return err
		}
		return nil
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode)
	if err != nil {
		if os.IsNotExist(err) {
			dir, _ := filepath.Split(filePath)
			if dir != "" {
				err := os.MkdirAll(dir, 0700)
				if err != nil {
					return err
				}
			}
			file, err = os.Create(filePath)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	n, err := io.Copy(file, fileContent)
	if err != nil {
		return err
	}
	if n != f.Size {
		return fmt.Errorf("file content size is not equal with fileinfo.size")
	}

	err = file.Chmod(f.Mode)
	if err != nil {
		return err
	}
	err = os.Chtimes(filePath, time.Now(), f.ModTime)
	if err != nil {
		return err
	}
	return nil
}
