package osext

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var s3Client *s3.Client

func getS3(ctx context.Context) (*s3.Client, error) {
	if s3Client != nil {
		return s3Client, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
	if err != nil {
		return nil, err
	}

	s3Client = s3.NewFromConfig(cfg)
	return s3Client, nil
}

func bucketKeyFromUrl(u *url.URL) (bucket, key string) {
	bucket = u.Host
	key = strings.TrimLeft(u.Path, "/")
	return
}

func Create(ctx context.Context, rawUrl string, r io.Reader) error {
	logger := logging.Get(ctx)

	u, err := url.Parse(rawUrl)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "s3":
		s3c, err := getS3(ctx)
		if err != nil {
			return err
		}

		bucket, key := bucketKeyFromUrl(u)
		logger, ctx = logging.WithAttrs(ctx, "bucket", bucket, "key", key)

		var buf bytes.Buffer
		rh := common.ReaderSHA256(r)
		n, err := io.Copy(&buf, rh)

		logger.Debug("putting object", "bytes", n, "SHA256", rh.HexDigest())
		o, err := s3c.PutObject(ctx, &s3.PutObjectInput {
			Bucket: aws.String(bucket),
			Key: aws.String(key),
			Body: &buf,
			ChecksumSHA256: aws.String(rh.B64Digest()),
		})
		if err == nil {
			logger.Debug("put object", "VersionId", aws.ToString(o.VersionId), "SHA256", rh.B64Digest())
		}
		return err
	case "http", "https":
		// TODO
		return fmt.Errorf("unimplemented URL scheme: %s", u.Scheme)
	case "", "file":
		path := filepath.Join(u.Host, u.Path)

		logger, _ = logging.WithAttrs(ctx, "path", path)

		logger.Debug("create")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		wh := common.WriterSHA256(f)

		n, err := io.Copy(wh, r)
		if err != nil {
			return err
		}
		if err == nil {
			logger.Debug("wrote", "bytes", n, "SHA256", wh.HexDigest())
		}
		return err
	default:
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
}

func IsNotExist(err error) bool {
	if os.IsNotExist(err) {
		return true
	}

	var apiError smithy.APIError
	if errors.As(err, &apiError) {
		switch apiError.(type) {
		case *types.NoSuchKey:
			return true
		}
	}

	return false
}

func Open(ctx context.Context, rawUrl string) (io.ReadCloser, error) {
	logger := logging.Get(ctx)

	u, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "s3":
		s3c, err := getS3(ctx)
		if err != nil {
			return nil, err
		}

		bucket, key := bucketKeyFromUrl(u)
		logger, ctx = logging.WithAttrs(ctx, "bucket", bucket, "key", key)

		logger.Debug("get object")
		o, err := s3c.GetObject(ctx, &s3.GetObjectInput {
			Bucket: aws.String(bucket),
			Key: aws.String(key),
		})

		if err != nil {
			return nil, err
		}

		logger.Debug("get object successful", "VersionId", aws.ToString(o.VersionId))

		return o.Body, nil
	case "http", "https":
		rsp, err := http.Get(u.String())
		if err != nil {
			return nil, err
		}
		return rsp.Body, err
	case "", "file":
		path := filepath.Join(u.Host, u.Path)

		logger, _ = logging.WithAttrs(ctx, "path", path)

		logger.Debug("open")
		return os.Open(path)
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
}
