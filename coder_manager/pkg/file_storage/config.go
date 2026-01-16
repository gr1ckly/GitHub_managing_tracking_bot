package file_storage

type Config struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	Region         string
	Bucket         string
	ForcePathStyle bool
	MaxSizeBytes   int64
	PartSizeBytes  int64
	Concurrency    int
}
