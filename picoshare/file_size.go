package picoshare

import "errors"

var (
	ErrEmptyFile        = errors.New("filename must be non-empty")
	ErrNegativeFileSize = errors.New("file size must be positive")
)

type FileSize struct {
	size uint64
}

func FileSizeFromInt(val int) (FileSize, error) {
	return FileSizeFromInt64(int64(val))
}

func FileSizeFromInt64(val int64) (FileSize, error) {
	if val < 0 {
		return FileSize{}, ErrNegativeFileSize
	}

	return FileSizeFromUint64(uint64(val))
}

func FileSizeFromUint64(val uint64) (FileSize, error) {
	if val == 0 {
		return FileSize{}, ErrEmptyFile
	}

	return FileSize{
		size: uint64(val),
	}, nil
}

func (fileSize FileSize) Equal(o FileSize) bool {
	return fileSize.size == o.size
}

func (fileSize FileSize) UInt64() uint64 {
	return fileSize.size
}
