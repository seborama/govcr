package fileio_test

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

func makeS3ClientWithBucket(bucketName string) (*s3.Client, error) {
	if err := godotenv.Load("../.envrc"); err != nil {
		// NOTE: this is non-fatal: the environment may already be set correctly.
		slog.Warn(".envrc load failed: ", slog.String("error", err.Error()))
	}

	ctx := context.Background()

	awsEndpoint := os.Getenv("AWS_ENDPOINT_URL")
	awsRegion := os.Getenv("AWS_DEFAULT_REGION")

	// Load the default AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(awsRegion),
	)
	if err != nil {
		panic("Cannot load the AWS configs: " + err.Error())
	}

	// UsePathStyle is required for local S3 emulators on localhost.
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(awsEndpoint)
	})

	if err = createBucket(ctx, s3Client, awsRegion, bucketName); err != nil {
		return nil, errors.WithStack(err)
	}

	return s3Client, nil
}

func createBucket(ctx context.Context, s3Client *s3.Client, region, name string) error {
	_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &name,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		},
	})
	return err
}
