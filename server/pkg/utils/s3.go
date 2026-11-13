package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type ObjectManager struct {
	client *s3.Client
	bucket string
}

func NewObjectManager(ctx context.Context) (*ObjectManager, error) {
	endpoint := os.Getenv("GARAGE_ENDPOINT")
	accessKey := os.Getenv("GARAGE_ACCESS_KEY")
	secretKey := os.Getenv("GARAGE_SECRET_KEY")
	region := os.Getenv("GARAGE_REGION")
	bucket := "logs"

	//hacky fix this later
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("Garage endpoint and credentials are required")
	}
	if region == "" {
		region = "garage"
	}

	config, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				accessKey,
				secretKey,
				"",
			),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(config, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	return &ObjectManager{client: client, bucket: bucket}, nil
}

func (om *ObjectManager) Upload(ctx context.Context, key string, body io.Reader, path string) error {
	checksum, err := fileChecksum(path)
	if err != nil {
		return err
	}

	_, err = om.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(om.bucket),
		Key:    aws.String(key),
		Body:   body,
		Metadata: map[string]string{
			"sha256": checksum,
		},
	})

	if err != nil {
		return fmt.Errorf("upload object error %q: %w", key, err)
	}

	return nil
}

func (om *ObjectManager) Download(ctx context.Context, key string) ([]byte, error) {
	out, err := om.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(om.bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return nil, fmt.Errorf("download object %q: %w", key, err)
	}
	defer out.Body.Close()

	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object body %q: %w", key, err)
	}

	return data, nil
}

func (om *ObjectManager) RemoteChecksum(ctx context.Context, key string) (checksum string, found bool, err error) {
	out, err := om.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(om.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("head object %q: %w", key, err)
	}
	return out.Metadata["sha256"], true, nil
}

func fileChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
