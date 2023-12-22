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
	"compress/gzip"
	"strconv"

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

	ignoreMissingFlag := flag.Bool("ignore-missing", common.GetenvBool("IGNORE_MISSING"), "ignore missing files")
	tarballNotExistOkFlag := flag.Bool("tarball-not-exist-ok", common.GetenvBool("NOT_EXIST_OK"), "fail gracefully if tarball does not exist")

	gzipFlag := flag.String("gzip", common.Getenv("GZIP"), "compress using gzip level")

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
		if err != nil {
			log.Fatal(err)
		}
		logger.Debug("root", "path", root)
	} else {
		root, err = filepath.Abs(root)
		if err != nil {
			log.Fatal(err)
		}
		logger.Info("root", "path", root)
	}

	var m *manifest.Manifest
	if *manifestFlag == "" {
		m = &manifest.Manifest{}
	} else {
		m, err = manifest.Load(ctx, *manifestFlag, root)
		if err != nil {
			logger.Error("unable to load manifest", "manifest", *manifestFlag, "err", err)
			os.Exit(1)
		}
	}
	m.IgnoreMissing = *ignoreMissingFlag

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
			fmt.Fprint(os.Stderr, "more than one action specified")
			os.Exit(2)
		}
		action = ActionCreate
		tarball = *createFlag
	}
	if *extractFlag != "" {
		if action != ActionNoop {
			fmt.Fprint(os.Stderr, "more than one action specified")
			os.Exit(2)
		}
		action = ActionExtract
		tarball = *extractFlag
	}

	logger, ctx = logging.WithAttrs(ctx, "tarball", tarball)

	gzipLevel := gzip.NoCompression
	if *gzipFlag != "" {
		gzipLevel, err = strconv.Atoi(*gzipFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "unable to parse as integer: %s (%v)", *gzipFlag, err)
			os.Exit(2)
		}
	} else if strings.HasSuffix(tarball, ".gz") || strings.HasSuffix(tarball, ".tgz") {
		gzipLevel = gzip.DefaultCompression
	}

	switch action {
	case ActionCreate:
		var buf bytes.Buffer
		if err := m.Create(ctx, &buf); err != nil {
			logger.Error("unable to create tarball", "err", err)
			os.Exit(1)
		}

		r := io.Reader(&buf)
		if gzipLevel != gzip.NoCompression {
			var cmp bytes.Buffer
			w, err := gzip.NewWriterLevel(&cmp, gzipLevel)
			if err != nil {
				logger.Error("unable to initialize gzip", "err", err)
				os.Exit(1)
			}

			n, err := io.Copy(w, r)
			if err != nil {
				logger.Error("unable to write gzip", "err", err)
				os.Exit(1)
			}

			w.Close()

			logger.Debug("gzip", "level", gzipLevel, "original", n, "compressed", cmp.Len())

			r = io.Reader(&cmp)
		}

		rh := common.ReaderSHA256(r)

		if err := Create(ctx, tarball, rh); err != nil {
			logger.Error("unable to write tarball", "err", err)
			os.Exit(1)
		}

		logger.Info("created", "SHA256", rh.HexDigest())
	case ActionExtract:
		logger.Info("extracting")
		f, err := Open(ctx, tarball)
		if IsNotExist(err) && *tarballNotExistOkFlag {
			logger.Info("failing gracefully: does not exist", "tarball", tarball)
			break
		}
		if err != nil {
			logger.Error("unable to open tarball", "err", err)
			os.Exit(1)
		}
		defer f.Close()

		rh := common.ReaderSHA256(f)
		r := io.Reader(rh)
		if gzipLevel != gzip.NoCompression {
			g, err := gzip.NewReader(r)
			if err != nil {
				logger.Error("unable to initialize gzip", "err", err)
				os.Exit(1)
			}
			defer g.Close()
			r = g
		}

		if err := m.Extract(ctx, r); err != nil {
			logger.Error("unable to extract tarball", "err", err)
			os.Exit(1)
		}

		logger.Info("extracted", "SHA256", rh.HexDigest())
	case ActionNoop:
		logger.Info("noop")
	}
}
