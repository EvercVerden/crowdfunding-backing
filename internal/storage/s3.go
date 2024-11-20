package storage

import (
	"bytes"
	"fmt"
	"mime/multipart"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
	s3     *s3.S3
	bucket string
}

func NewS3Client(region, bucket string) (*S3Client, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	return &S3Client{
		s3:     s3.New(sess),
		bucket: bucket,
	}, nil
}

func (c *S3Client) UploadFile(file *multipart.FileHeader, path string) (string, error) {
	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	buffer := make([]byte, file.Size)
	_, err = f.Read(buffer)
	if err != nil {
		return "", err
	}

	_, err = c.s3.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(c.bucket),
		Key:           aws.String(path),
		Body:          bytes.NewReader(buffer),
		ContentLength: aws.Int64(file.Size),
		ContentType:   aws.String(file.Header.Get("Content-Type")),
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", c.bucket, path), nil
}
