package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"context"
	"io"
	"bytes"
	"strings"
	"fmt"
	"compress/gzip"
	"strconv"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/internal/manifest"
	"rootmos.io/sitepkg/sealedbox"
	"rootmos.io/sitepkg/osext"
)

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")

	createFlag := flag.String("create", common.Getenv("CREATE"), "write tarball")
	extractFlag := flag.String("extract", common.Getenv("EXTRACT"), "extract tarball")
	// verifyFlag := flag.String("verify", common.Getenv("VERIFY"), "verify tarball") // or status? check? test?

	ignoreMissingFlag := flag.Bool("ignore-missing", common.GetenvBool("IGNORE_MISSING"), "ignore missing files")
	tarballNotExistOkFlag := flag.Bool("tarball-not-exist-ok", common.GetenvBool("NOT_EXIST_OK"), "fail gracefully if tarball does not exist")

	gzipFlag := flag.String("gzip", common.Getenv("GZIP"), "compress using gzip level")

	keyfileFlag := flag.String("keyfile", common.Getenv("KEYFILE"), "encrypt/decrypt using the specified keyfile")
	awsSecretsmanagerSecretArnFlag := flag.String(
		"aws-secretsmanager-secret-arn",
		common.Getenv("AWS_SECRETSMANAGER_SECRET_ARN"),
		"encrypt/decrypt using the key fetched from AWS Secrets Manager Secret specified by its ARN",
	)

	logConfig := logging.PrepareConfig(common.EnvPrefix)

	flag.Parse()

	logger, err := logConfig.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug("hello")

	ctx := logging.Set(context.Background(), logger)

	logger.Tracef("foo: %d", 7)

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
	} else if strings.HasSuffix(tarball, ".gz") || strings.HasSuffix(tarball, ".gz.enc") || strings.HasSuffix(tarball, ".tgz") || strings.HasSuffix(tarball, ".tgz.enc") {
		gzipLevel = gzip.DefaultCompression
	}

	var key *sealedbox.Key
	if *keyfileFlag != "" || *awsSecretsmanagerSecretArnFlag != "" {
		if *keyfileFlag != "" && *awsSecretsmanagerSecretArnFlag != "" {
			fmt.Fprint(os.Stderr, "both keyfile and AWS Secrets Manager Secret specified")
			os.Exit(2)
		}

		if *keyfileFlag != "" {
			path := *keyfileFlag
			logger.Info("using keyfile", "keyfile", path)
			key, err = sealedbox.LoadKeyfile(path)
			if err != nil {
				logger.Error("unable to load keyfile", "keyfile", path, "err", err,)
				os.Exit(1)
			}
			defer key.Close()
		}

		if *awsSecretsmanagerSecretArnFlag != "" {
			key, err = getKeyFromSMSecretValue(ctx, *awsSecretsmanagerSecretArnFlag)
			if err != nil {
				os.Exit(1)
			}
			defer key.Close()
		}
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

		if key != nil {
			pt, err := io.ReadAll(r)
			if err != nil {
				logger.Error("unable to read tarball", "err", err)
				os.Exit(1)
			}

			box, err := sealedbox.Seal(key, pt)
			if err != nil {
				logger.Error("unable to encrypt tarball", "err", err)
				os.Exit(1)
			}

			enc, err := box.MarshalBinary()
			if err != nil {
				logger.Error("unable to marshal tarball", "err", err)
				os.Exit(1)
			}

			logger.Debug("encrypted")

			r = bytes.NewReader(enc)
		}

		rh := common.ReaderSHA256(r)

		if err := osext.Create(ctx, tarball, rh); err != nil {
			logger.Error("unable to write tarball", "err", err)
			os.Exit(1)
		}

		logger.Info("created", "SHA256", rh.HexDigest())
	case ActionExtract:
		logger.Info("extracting")
		f, err := osext.Open(ctx, tarball)
		if osext.IsNotExist(err) && *tarballNotExistOkFlag {
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

		if key != nil {
			bs, err := io.ReadAll(r)
			if err != nil {
				logger.Error("unable to read tarball", "err", err)
				os.Exit(1)
			}

			var box sealedbox.Box
			if err := box.UnmarshalBinary(bs); err != nil {
				logger.Error("unable to unmarshal box", "err", err)
				os.Exit(1)
			}

			pt, err := box.Open(key)
			if err != nil {
				logger.Error("unable to decrypt box", "err", err)
				os.Exit(1)
			}

			logger.Debug("decrypted")

			r = bytes.NewReader(pt)
		}

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
