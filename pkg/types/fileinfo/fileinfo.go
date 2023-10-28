package fileinfo

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	pb "github.com/quic-s/quics-protocol/proto/v1"
)

// FileInfo is a file metadata structure.
// It is used to send and receive file metadata through the connection.
// You can get this struct as a result when receiving a file through the connection.
type FileInfo struct {
	Name    string
	Size    int64
	Mode    os.FileMode
	ModTime time.Time
	IsDir   bool
}

// Create new FileInfo instance from os.FileInfo.
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

// Create new FileInfo instance from protobuf.
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

// Convert FileInfo to protobuf.
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

// WriteFileWithInfo writes the file with metadata to the disk.
// The file path and file data(io.Reader type) need to be passed as parameters.
// This method creates a directory if the directory does not exist or received file is directory.
// If the file already exists, it will be overwritten.
func (f *FileInfo) WriteFileWithInfo(filePath string, fileContent io.Reader) error {
	// When the file is a directory, create the directory and return.
	if f.IsDir {
		err := os.MkdirAll(filePath, f.Mode)
		if err != nil {
			return err
		}
		file, err := os.Open(filePath)
		if err != nil {
			return err
		}

		// Set file metadata.
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

	// When the file is not a directory, create the file and write the file content.

	// Open file with O_TRUNC flag to overwrite the file when the file already exists.
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode)
	if err != nil {
		// If the file does not exist, create the file.
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

	// Write file content.
	n, err := io.Copy(file, fileContent)
	if err != nil {
		return err
	}
	if n != f.Size {
		return errors.New("file content size is not equal with fileinfo.size")
	}

	// Set file metadata.
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
