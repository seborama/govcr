package fileio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
)

type S3File struct {
	s3Client *s3.Client // TODO: change this to an interface once the methods are known.
}

type S3Client interface{}

func NewAWS(s3Client *s3.Client) *S3File {
	return &S3File{
		s3Client: s3Client,
	}
}

func (f *S3File) MkdirAll(path string, perm os.FileMode) error {
	// this is a noop in S3
	return nil
}

func (f *S3File) ReadFile(name string) ([]byte, error) {
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
func (f *S3File) WriteFile(name string, data []byte, _ os.FileMode) error {
	bucket, key, err := f.bucketAndKey(name)
	if err != nil {
		return err
	}
	fmt.Println("bucket:", bucket)
	fmt.Println("key:", key)

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

func (f *S3File) IsNotExist(err error) bool {
	panic("not implemented") // TODO: Implement
}

func (f *S3File) bucketAndKey(name string) (bucket, key string, err error) {
	splits := strings.SplitN(name, "/", 3)
	if len(splits) != 3 {
		err = errors.Errorf("invalid S3 object name: '%s' - expected format is '/bucket/[folder/.../]file'", name)
		return
	}

	bucket = splits[1]
	key = splits[2]

	return
}
