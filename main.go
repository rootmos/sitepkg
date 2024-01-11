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
	"compress/gzip"
	"strconv"
	"fmt"

	"rootmos.io/go-utils/hashed"
	"rootmos.io/go-utils/logging"
	"rootmos.io/go-utils/osext"
	"rootmos.io/go-utils/sealedbox"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/manifest"
)


type state struct {
	tarball string
	m *manifest.Manifest
	key *sealedbox.Key
	gzipLevel int
	tarballNotExistOk bool
}

func (st *state) create(ctx context.Context) error {
	logger := logging.Get(ctx)

	var buf bytes.Buffer
	if err := st.m.Create(ctx, &buf); err != nil {
		return err
	}

	r := io.Reader(&buf)
	if st.gzipLevel != gzip.NoCompression {
		var cmp bytes.Buffer
		w, err := gzip.NewWriterLevel(&cmp, st.gzipLevel)
		if err != nil {
			return fmt.Errorf("unable to initialize gzip: %s", err)
		}

		n, err := io.Copy(w, r)
		if err != nil {
			return fmt.Errorf("unable to write gzip: %s", err)
		}

		w.Close()

		logger.Debug("gzip", "level", st.gzipLevel, "original", n, "compressed", cmp.Len())

		r = io.Reader(&cmp)
	}

	if st.key != nil {
		pt, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("unable to read tarball: %v", err)
		}

		box, err := sealedbox.Seal(st.key, pt)
		if err != nil {
			return fmt.Errorf("unable to encrypt tarball: %v", err)
		}

		enc, err := box.MarshalBinary()
		if err != nil {
			return fmt.Errorf("unable to marshal tarball: %v", err)
		}

		logger.Debug("encrypted")

		r = bytes.NewReader(enc)
	}

	rh := hashed.ReaderSHA256(r)

	if err := osext.Create(ctx, st.tarball, rh); err != nil {
		return fmt.Errorf("unable to write tarball: %v", err)
	}

	logger.Info("created", "SHA256", rh.HexDigest())

	return nil
}

func (st *state) extract(ctx context.Context) error {
	logger := logging.Get(ctx)

	logger.Info("extracting")
	f, err := osext.Open(ctx, st.tarball)
	if osext.IsNotExist(err) && st.tarballNotExistOk {
		logger.Info("failing gracefully: tarball does not exist", "tarball", st.tarball)
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to open tarball: %v", err)
	}
	defer f.Close()

	rh := hashed.ReaderSHA256(f)
	r := io.Reader(rh)

	if st.key != nil {
		bs, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("unable to read tarball: %s", err)
		}

		var box sealedbox.Box
		if err := box.UnmarshalBinary(bs); err != nil {
			return fmt.Errorf("unable to unmarshal box: %s", err)
		}

		pt, err := box.Open(st.key)
		if err != nil {
			return fmt.Errorf("unable to decrypt box: %s", err)
		}

		logger.Debug("decrypted")

		r = bytes.NewReader(pt)
	}

	if st.gzipLevel != gzip.NoCompression {
		g, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("unable to initialize gzip: %s", err)
		}
		defer g.Close()
		r = g
	}

	if err := st.m.Extract(ctx, r); err != nil {
		return fmt.Errorf("unable to extract tarball: %s", err)
	}

	logger.Info("extracted", "SHA256", rh.HexDigest())
	return nil
}

func filenameSuggestCompression(path string) bool {
	return (strings.HasSuffix(path, ".gz") ||
		strings.HasSuffix(path, ".gz.enc") ||
		strings.HasSuffix(path, ".tgz") ||
		strings.HasSuffix(path, ".tgz.enc"))
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

	keyfileFlag := flag.String("keyfile", common.Getenv("KEYFILE"), "encrypt/decrypt using the specified keyfile")
	awsSecretsmanagerSecretArnFlag := flag.String(
		"aws-secretsmanager-secret-arn",
		common.Getenv("AWS_SECRETSMANAGER_SECRET_ARN"),
		"encrypt/decrypt using the key fetched from AWS Secrets Manager Secret specified by its ARN",
	)

	logConfig := logging.PrepareConfig(common.EnvPrefix)

	flag.Parse()

	logger, closer, err := logConfig.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer closer()
	logger.Debug("hello")

	ctx := logging.Set(context.Background(), logger)

	st := state {
		tarballNotExistOk: *tarballNotExistOkFlag,
	}

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

	if *manifestFlag == "" {
		st.m = &manifest.Manifest{}
	} else {
		path := *manifestFlag
		st.m, err = manifest.Load(ctx, path, root)
		if err != nil {
			logger.With("err", err).ExitfContext(ctx, 1, "unable to load manifest: %s", path)
		}
	}
	st.m.IgnoreMissing = *ignoreMissingFlag

	for _, p := range flag.Args() {
		st.m.Add(p)
	}

	const (
		ActionNoop = iota
		ActionCreate
		ActionExtract
	)
	action := ActionNoop

	if *createFlag != "" {
		if action != ActionNoop {
			logger.ExitContext(ctx, 2, "more than one action specified")
		}
		action = ActionCreate
		st.tarball = *createFlag
	}
	if *extractFlag != "" {
		if action != ActionNoop {
			logger.ExitContext(ctx, 2, "more than one action specified")
		}
		action = ActionExtract
		st.tarball = *extractFlag
	}

	logger, ctx = logging.WithAttrs(ctx, "tarball", st.tarball)

	st.gzipLevel = gzip.NoCompression
	if *gzipFlag != "" {
		st.gzipLevel, err = strconv.Atoi(*gzipFlag)
		if err != nil {
			logger.With("err", err).ExitfContext(ctx, 2, "unable to parse as integer: %s", *gzipFlag)
		}
	} else if filenameSuggestCompression(st.tarball) {
		st.gzipLevel = gzip.DefaultCompression
	}

	if *keyfileFlag != "" || *awsSecretsmanagerSecretArnFlag != "" {
		if *keyfileFlag != "" && *awsSecretsmanagerSecretArnFlag != "" {
			logger.ExitContext(ctx, 2, "both keyfile and AWS Secrets Manager Secret specified")
		}

		if *keyfileFlag != "" {
			path := *keyfileFlag
			logger.Info("using keyfile", "keyfile", path)
			st.key, err = sealedbox.LoadKeyfile(path)
			if err != nil {
				logger.With("err", err).ExitfContext(ctx, 1, "unable to load keyfile: %s", path)
			}
			defer st.key.Close()
		}

		if *awsSecretsmanagerSecretArnFlag != "" {
			arn := *awsSecretsmanagerSecretArnFlag
			st.key, err = getKeyFromSMSecretValue(ctx, arn)
			if err != nil {
				logger.With("err", err).ExitfContext(ctx, 1, "unable to get key from Secrets Manager: %s", arn)
			}
			defer st.key.Close()
		}
	}

	switch action {
	case ActionCreate:
		if err := st.create(ctx); err != nil {
			logger.Exit(1, "unable to create tarball: %v", err)
		}
	case ActionExtract:
		if err := st.extract(ctx); err != nil {
			logger.Exit(1, "unable to extract tarball: %v", err)
		}
	case ActionNoop:
		logger.Info("noop")
	}
}
