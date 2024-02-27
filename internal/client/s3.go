package client

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client *s3.Client
}

func NewS3Client(cfg aws.Config) (*S3Client, error) {
	s3Client := s3.NewFromConfig(cfg)
	return &S3Client{client: s3Client}, nil
}

func (c *S3Client) PutObject(bucket, key string, data []byte) error {
	_, err := c.client.PutObject(context.Background(),
		&s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
			Body:   bytes.NewReader(data),
		},
	)
	if err != nil {
		return fmt.Errorf("error putting object: %w", err)
	}

	return nil
}
