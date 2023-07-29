package fileio_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v13/fileio"
)

func TestS3Client(t *testing.T) {
	if err := godotenv.Load("../.envrc"); err != nil {
		log.Println(".envrc load failed: ", err)
	}

	ctx := context.Background()

	awsEndpoint := os.Getenv("LOCALSTACK_ENDPOINT")
	awsRegion := os.Getenv("AWS_DEFAULT_REGION")
	bucketName := "tests3client-writefile-" + uuid.New().String() // warning: max length: 63 chars
	log.Println("bucketName:", bucketName)

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

	// Note: "UsePathStyle" REQUIRED for localstack
	// https://docs.localstack.cloud/user-guide/aws/s3/
	// https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html
	s3Client := s3.NewFromConfig(sdkConfig, func(o *s3.Options) { o.UsePathStyle = true })
	err = createBucket(ctx, s3Client, awsRegion, bucketName)
	require.NoError(t, err)

	s3f := fileio.NewAWS(s3Client)

	objectName := "/" + bucketName + "/Development/TestS3Client_WriteFile.tmp"

	// NotExist true
	notExist, err := s3f.NotExist(objectName)
	require.True(t, notExist)
	require.NoError(t, err)

	// WriteFile
	err = s3f.WriteFile(objectName, []byte("hello"), 0)
	require.NoError(t, err)

	// NotExist false
	notExist, err = s3f.NotExist(objectName)
	require.False(t, notExist)
	require.NoError(t, err)

	// ReadFile
	data, err := s3f.ReadFile(objectName)
	require.NoError(t, err)
	require.EqualValues(t, "hello", data)
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
