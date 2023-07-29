package fileio

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
)

type S3Storage struct {
	s3Client *s3.Client // TODO: change this to an interface once the methods are known.
}

func NewAWS(s3Client *s3.Client) *S3Storage {
	return &S3Storage{
		s3Client: s3Client,
	}
}

func (f *S3Storage) MkdirAll(path string, perm os.FileMode) error {
	// this is a noop in S3
	return nil
}

func (f *S3Storage) ReadFile(name string) ([]byte, error) {
	bucket, key, err := f.bucketAndKey(name)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	result, err := f.s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return data, errors.WithStack(err)
}

// TODO: instead of `data []byte`, we could use an io.Writer
// TODO: use an options style functional param instead of `_ os.FileMode`
func (f *S3Storage) WriteFile(name string, data []byte, _ os.FileMode) error {
	bucket, key, err := f.bucketAndKey(name)
	if err != nil {
		return err
	}

	largeBuffer := bytes.NewReader(data)
	const partSize int64 = 10 * 1024 * 1024
	uploader := manager.NewUploader(f.s3Client, func(u *manager.Uploader) {
		u.PartSize = partSize
	})
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   largeBuffer,
	})

	return errors.WithStack(err)
}

func (f *S3Storage) NotExist(name string) (bool, error) {
	exists, err := f.exists(context.Background(), name)
	return !exists, err
}

func (f *S3Storage) bucketAndKey(name string) (bucket, key string, err error) {
	splits := strings.SplitN(name, "/", 3)
	if len(splits) != 3 {
		err = errors.Errorf("invalid S3 object name: '%s' - expected format is '/bucket/[folder/.../]file'", name)
		return
	}

	bucket = splits[1]
	key = splits[2]

	return
}

func (f *S3Storage) exists(ctx context.Context, name string) (bool, error) {
	bucket, key, err := f.bucketAndKey(name)
	if err != nil {
		return false, errors.WithStack(err)
	}

	_, err = f.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	// \(`o')/ there must be an intelligent way to do this \(`o')/
	var oe *smithy.OperationError
	if errors.As(err, &oe) && oe.Err != nil {
		oeErrUW := errors.Unwrap(oe.Err)
		var nf *types.NotFound
		if errors.As(oeErrUW, &nf) {
			return false, nil
		}
	}

	return err == nil, errors.WithStack(err)
}
