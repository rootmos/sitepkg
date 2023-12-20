package main

import (
	"log"
	"log/slog"
	"flag"
	"os"
	"path/filepath"
	"bufio"
	"io"
	"archive/tar"
	"context"

	"rootmos.io/sitepkg/internal/common"
)

type Manifest struct {
	Paths []string
}

func Load(ctx context.Context, path string) (m *Manifest, err error) {
	logger, ctx := withLogger(ctx, func(l *slog.Logger) *slog.Logger {
		return l.With("manifest", path)
	})

	f, err := os.Open(path)
	defer f.Close()

	m = &Manifest{}

	s := bufio.NewScanner(f)
	for s.Scan() {
		p := s.Text()
		logger.Debug("adding path to manifest", "path", p)
		m.Paths = append(m.Paths, p)
	}
	if err = s.Err(); err != nil {
		return
	}

	return m, nil
}

func (m *Manifest) CreateTarball(ctx context.Context, w io.Writer) (err error) {
	logger := getLogger(ctx)

	t := tar.NewWriter(w)
	defer func() {
		err = t.Close()
	}()

	for _, p := range m.Paths {
		logger.Debug("adding path", "path", p)
	}

	return
}

var (
	logLevelFlag = flag.String("log", common.Getenv("LOG_LEVEL"), "set logging level")
)

func setupDefaultLogger() (*slog.Logger, error) {
	level := slog.LevelInfo
	if *logLevelFlag != "" {
		err := level.UnmarshalText([]byte(*logLevelFlag))
		if err != nil {
			return nil, err
		}
	}
	opts := slog.HandlerOptions{ Level: level }
	logger := slog.New(slog.NewTextHandler(os.Stderr, &opts))
	slog.SetDefault(logger)
	return logger, nil
}

func setLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}

func getLogger(ctx context.Context) *slog.Logger {
	return ctx.Value("logger").(*slog.Logger)
}

func withLogger(ctx context.Context, fs ...func(*slog.Logger) *slog.Logger) (*slog.Logger, context.Context) {
	logger := getLogger(ctx)
	for _, f := range fs {
		logger = f(logger)
	}
	return logger, setLogger(ctx, logger)
}

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")
	outputFlag := flag.String("output", common.Getenv("OUTPUT"), "write tarball to path")
	flag.Parse()

	logger, err := setupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug("hello")

	ctx := setLogger(context.Background(), logger)

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

	if *manifestFlag == "" {
		log.Fatal("manifest not specified")
	}

	m, err := Load(ctx, *manifestFlag)
	if err != nil {
		log.Fatal(err)
	}

	output := *outputFlag
	if output == "" {
		log.Fatal("output not specified")
	}

	logger.Info("creating tarball", "path", output)
	f, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, ctx = withLogger(ctx, func(l *slog.Logger) *slog.Logger {
		return l.With("tarball", output)
	})

	err = m.CreateTarball(ctx, f)
	if err != nil {
		log.Fatal(err)
	}
}
