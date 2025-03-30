package fileio_test

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/seborama/govcr/v16/fileio"
)

func TestS3Client(t *testing.T) {
	bucketName := "tests3client-writefile-" + uuid.New().String() // warning: max length: 63 chars
	slog.Info("AWS", slog.String("bucketName:", bucketName))

	s3Client, err := makeS3ClientWithBucket(bucketName)
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
