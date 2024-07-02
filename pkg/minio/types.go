package minio

type UploadMediaOptions struct {
	BucketName  string
	FilePath    string
	ObjectName  string
	ContentType string
}

type SaveFileOptions struct {
	BucketName  string
	ObjectName  string
	ContentType string
	FileData    []byte
}

type Bucket string

const (
	TemporaryBucket = Bucket("temporary-bucket")
)

func (b Bucket) String() string {
	return string(b)
}
