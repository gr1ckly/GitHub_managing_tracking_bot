package file_storage

import (
	"context"
	"errors"
	"io"
	"strings"

	internalfs "coder_manager/internal/file_storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Storage struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	maxSize  int64
}

func NewS3Storage(cfg Config) (*S3Storage, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	accessKey := strings.TrimSpace(cfg.AccessKey)
	secretKey := strings.TrimSpace(cfg.SecretKey)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "us-east-1"
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required")
	}
	forcePathStyle := cfg.ForcePathStyle

	var cfgOpts []func(*awsconfig.LoadOptions) error
	cfgOpts = append(cfgOpts, awsconfig.WithRegion(region))
	if accessKey != "" || secretKey != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	}
	if endpoint != "" {
		cfgOpts = append(cfgOpts, awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, _ ...interface{}) (aws.Endpoint, error) {
				if service == s3.ServiceID {
					return aws.Endpoint{
						URL:               endpoint,
						PartitionID:       "aws",
						SigningRegion:     region,
						HostnameImmutable: true,
					}, nil
				}
				return aws.Endpoint{}, &aws.EndpointNotFoundError{}
			},
		)))
		forcePathStyle = true
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), cfgOpts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = forcePathStyle
	})
	uploader := manager.NewUploader(client, func(u *manager.Uploader) {
		if cfg.PartSizeBytes > 0 {
			u.PartSize = cfg.PartSizeBytes
		}
		if cfg.Concurrency > 0 {
			u.Concurrency = cfg.Concurrency
		}
	})
	maxSize := cfg.MaxSizeBytes
	if maxSize == 0 {
		maxSize = 1 << 30
	}
	return &S3Storage{client: client, uploader: uploader, bucket: bucket, maxSize: maxSize}, nil
}

func (s *S3Storage) SaveFile(ctx context.Context, req internalfs.SaveFileRequest) error {
	if s.client == nil {
		return errors.New("s3 client is nil")
	}
	if req.Content == nil {
		return errors.New("file content is nil")
	}
	if strings.TrimSpace(req.Key) == "" {
		return errors.New("storage key is empty")
	}
	if s.uploader == nil {
		return errors.New("s3 uploader is nil")
	}
	if req.Size != nil && *req.Size > s.maxSize {
		return errors.New("file exceeds size limit")
	}
	body := req.Content
	if req.Size == nil {
		body = newLimitedReader(req.Content, s.maxSize)
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(req.Key),
		Body:   body,
	}
	_, err := s.uploader.Upload(ctx, input)
	return err
}

func (s *S3Storage) DownloadFile(ctx context.Context, req internalfs.DownloadFileRequest) (io.ReadCloser, *int64, error) {
	if s.client == nil {
		return nil, nil, errors.New("s3 client is nil")
	}
	key := strings.TrimSpace(req.Key)
	if key == "" {
		return nil, nil, errors.New("storage key is empty")
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}
	output, err := s.client.GetObject(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	size := output.ContentLength
	if size != nil && *size > s.maxSize {
		_ = output.Body.Close()
		return nil, nil, errors.New("file exceeds size limit")
	}
	reader := io.ReadCloser(output.Body)
	if size == nil || *size <= 0 {
		reader = newLimitedReadCloser(output.Body, s.maxSize)
	}
	return reader, size, nil
}

type limitedReader struct {
	reader io.Reader
	limit  int64
	read   int64
}

func newLimitedReader(reader io.Reader, limit int64) io.Reader {
	return &limitedReader{reader: reader, limit: limit}
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.read >= l.limit {
		return 0, errors.New("file exceeds size limit")
	}
	if int64(len(p)) > l.limit-l.read {
		p = p[:l.limit-l.read]
	}
	n, err := l.reader.Read(p)
	l.read += int64(n)
	if l.read >= l.limit && err == nil {
		return n, errors.New("file exceeds size limit")
	}
	return n, err
}

type limitedReadCloser struct {
	reader io.ReadCloser
	limit  int64
	read   int64
}

func newLimitedReadCloser(reader io.ReadCloser, limit int64) io.ReadCloser {
	return &limitedReadCloser{reader: reader, limit: limit}
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	if l.read >= l.limit {
		return 0, errors.New("file exceeds size limit")
	}
	if int64(len(p)) > l.limit-l.read {
		p = p[:l.limit-l.read]
	}
	n, err := l.reader.Read(p)
	l.read += int64(n)
	if l.read >= l.limit && err == nil {
		return n, errors.New("file exceeds size limit")
	}
	return n, err
}

func (l *limitedReadCloser) Close() error {
	return l.reader.Close()
}
