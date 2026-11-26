package s3helper

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GeneratePresignedURL(
	s3Client *s3.Client,
	bucket,
	key string,
	expireTime time.Duration,
) (string, error) {
	preSign := s3.NewPresignClient(s3Client)
	preSignReq, err := preSign.PresignGetObject(
		context.Background(),
		&s3.GetObjectInput{
			Bucket: &bucket,
			Key:    &key,
		},
		s3.WithPresignExpires(expireTime),
	)
	if err != nil {
		return "", fmt.Errorf("Cannot generate a presigned HTTP Request: %w", err)
	}

	return preSignReq.URL, nil
}
