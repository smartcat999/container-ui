package storage

import (
	"io"
)

// Storage 定义仓库存储接口
type Storage interface {
	// 仓库操作
	ListRepositories() ([]string, error)

	// 标签操作
	ListTags(repository string) ([]string, error)

	// 清单操作
	GetManifest(repository, reference string) ([]byte, string, error)
	GetManifestByDigest(repository, digest string) ([]byte, string, error)
	PutManifest(repository, reference, digest string, manifest []byte) error
	DeleteManifest(repository, reference string) error

	// Blob 操作
	GetBlobSize(repository, digest string) (int64, error)
	GetBlob(repository, digest string) (io.ReadCloser, int64, error)
	DeleteBlob(repository, digest string) error

	// 上传操作
	InitiateUpload(repository, uploadID string) error
	AppendToUpload(repository, uploadID string, data []byte) (int64, error)
	CompleteUpload(repository, uploadID, digest string, data []byte) error
}
