package fileio_test

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
	"github.com/seborama/govcr/v13/fileio"
	"github.com/stretchr/testify/require"
)

func TestS3Client_WriteFile(t *testing.T) {
	if err := godotenv.Load("../.envrc"); err != nil {
		panic(err)
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
	require.NoError(t, err)

	s3Client := s3.NewFromConfig(sdkConfig, func(o *s3.Options) { o.UsePathStyle = true /* REQUIRED for localstack */ })
	if err = deleteBucket(ctx, s3Client, "blahdiblah"); err != nil {
		panic(err)
	}

	s3f := fileio.NewAWS(s3Client)
	err = s3f.WriteFile("/seborama-govcr/Development/TestS3Client_WriteFile.tmp", []byte("hello"), 0)
	require.NoError(t, err)
}

func createBucket(ctx context.Context, s3Client *s3.Client, name string) error {
	_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{})
	return err
}

func deleteBucket(ctx context.Context, s3Client *s3.Client, name string) error {
	_, err := s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &name,
	})

	// apologies: this has to be a mistake, there must be an intelligent way to do this...
	var oe *smithy.OperationError
	if errors.As(err, &oe) && oe.Err != nil {
		oeErrUW := errors.Unwrap(oe.Err)
		var gae *smithy.GenericAPIError
		if errors.As(oeErrUW, &gae) {
			if gae.ErrorCode() == (&types.NoSuchBucket{}).ErrorCode() {
				return nil
			}
		}
	}

	return err
}
