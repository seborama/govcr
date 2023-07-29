package examples_test

import (
	"context"
	"log"
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
		log.Println(".envrc load failed: ", err)
	}

	ctx := context.Background()

	awsEndpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	awsRegion := os.Getenv("AWS_DEFAULT_REGION")

	var optFns []func(*config.LoadOptions) error

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if awsEndpoint != "" {
			return aws.Endpoint{
				PartitionID:   "aws",
				URL:           awsEndpoint,
				SigningRegion: awsRegion,
			}, nil
		}
		// returning EndpointNotFoundError will allow the service to fallback to it's default resolution
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})
	optFns = append(optFns, config.WithEndpointResolverWithOptions(customResolver), config.WithRegion(awsRegion))

	sdkConfig, err := config.LoadDefaultConfig(ctx, optFns...)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Note: "UsePathStyle" REQUIRED for localstack
	// https://docs.localstack.cloud/user-guide/aws/s3/
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
	s3Client := s3.NewFromConfig(sdkConfig, func(o *s3.Options) { o.UsePathStyle = true })

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
