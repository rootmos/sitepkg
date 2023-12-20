package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"context"
	"io"
	"net/url"
	"bytes"
	"strings"
	"fmt"
	"errors"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/internal/manifest"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var s3Client *s3.Client

func S3(ctx context.Context) (*s3.Client, error) {
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
		s3c, err := S3(ctx)
		if err != nil {
			return err
		}

		bucket, key := bucketKeyFromUrl(u)
		logger, ctx = logging.WithAttrs(ctx, "bucket", bucket, "key", key)

		logger.Debug("put object")
		o, err := s3c.PutObject(ctx, &s3.PutObjectInput {
			Bucket: aws.String(bucket),
			Key: aws.String(key),
			Body: r,
		})
		if err == nil {
			logger.Debug("put object successful", "VersionId", aws.ToString(o.VersionId))
		}
		return err
	case "": fallthrough
	case "file":
		path := filepath.Join(u.Host, u.Path)

		logger, _ = logging.WithAttrs(ctx, "path", path)

		logger.Debug("create")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		n, err := io.Copy(f, r)
		if err != nil {
			return err
		}
		if err == nil {
			logger.Debug("wrote", "bytes", n)
		}
		return err
	default:
		return fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
}

func IsS3NoSuchKey(err error) bool {
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
		s3c, err := S3(ctx)
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
	case "": fallthrough
	case "file":
		path := filepath.Join(u.Host, u.Path)

		logger, _ = logging.WithAttrs(ctx, "path", path)

		logger.Debug("open")
		return os.Open(path)
	default:
		return nil, fmt.Errorf("unsupported URL scheme: %s", u.Scheme)
	}
}

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")

	createFlag := flag.String("create", common.Getenv("CREATE"), "write tarball")
	extractFlag := flag.String("extract", common.Getenv("EXTRACT"), "extract tarball")
	// verifyFlag := flag.String("verify", common.Getenv("VERIFY"), "verify tarball") // or status? check? test?

	noExistsOkFlag := flag.Bool("no-exists-ok", common.GetenvBool("NO_EXISTS_OK"), "fail gracefully if tarball does not exist")

	flag.Parse()

	logger, err := logging.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug("hello")

	ctx := logging.Set(context.Background(), logger)

	root := *chrootFlag
	if root == "" {
		root, err = os.Getwd()
	} else {
		root, err = filepath.Abs(root)
	}
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("chroot", "path", root)

	var m *manifest.Manifest
	if *manifestFlag == "" {
		m = &manifest.Manifest{}
	} else {
		m, err = manifest.Load(ctx, *manifestFlag, root)
		if err != nil {
			log.Fatal(err)
		}
	}

	for _, p := range flag.Args() {
		m.Add(p)
	}

	const (
		ActionNoop = iota
		ActionCreate
		ActionExtract
	)
	action := ActionNoop

	var tarball string
	if *createFlag != "" {
		if action != ActionNoop {
			log.Fatal("more than one action specified")
		}
		action = ActionCreate
		tarball = *createFlag
	}
	if *extractFlag != "" {
		if action != ActionNoop {
			log.Fatal("more than one action specified")
		}
		action = ActionExtract
		tarball = *extractFlag
	}

	logger, ctx = logging.WithAttrs(ctx, "tarball", tarball)

	switch action {
	case ActionCreate:
		var buf bytes.Buffer
		if err := m.Create(ctx, &buf); err != nil {
			log.Fatal(err)
		}

		if err := Create(ctx, tarball, &buf); err != nil {
			log.Fatal(err)
		}
	case ActionExtract:
		logger.Info("extracting")
		f, err := Open(ctx, tarball)
		if IsS3NoSuchKey(err) && *noExistsOkFlag {
			logger.Info("gracefully failing: no such key")
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if err := m.Extract(ctx, f); err != nil {
			log.Fatal(err)
		}
	case ActionNoop:
	}
}
