package stages

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config is the connection surface for the S3-compatible storage backend.
type S3Config struct {
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
}

// Upload walks outputDir and PutObjects every file into the bucket under the
// path <projectSlug>/<version>/<relative-path>, which is the storage contract
// shared by all doc-thor modules.
func Upload(cfg S3Config, projectSlug, version, outputDir string) error {
	s3Client := s3.NewFromConfig(aws.Config{
		Region: cfg.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.AccessKey, cfg.SecretKey, "",
		),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true
	})

	return filepath.WalkDir(outputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(outputDir, path)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", rel, err)
		}
		defer f.Close()

		key := projectSlug + "/" + version + "/" + filepath.ToSlash(rel)
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if _, err := s3Client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket:      aws.String(cfg.Bucket),
			Key:         aws.String(key),
			Body:        f,
			ContentType: aws.String(contentType),
		}); err != nil {
			return fmt.Errorf("put %s: %w", key, err)
		}

		return nil
	})
}
