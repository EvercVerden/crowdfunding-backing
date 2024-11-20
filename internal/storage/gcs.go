package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type GCSClient struct {
	client     *storage.Client
	bucketName string
}

func NewGCSClient(projectID, bucketName, credentialsFile string) (*GCSClient, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, err
	}

	return &GCSClient{
		client:     client,
		bucketName: bucketName,
	}, nil
}

func (c *GCSClient) UploadFile(file *multipart.FileHeader, path string) (string, error) {
	ctx := context.Background()
	bucket := c.client.Bucket(c.bucketName)
	obj := bucket.Object(path)

	writer := obj.NewWriter(ctx)
	defer writer.Close()

	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	if _, err = io.Copy(writer, src); err != nil {
		return "", err
	}

	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, path), nil
}
