package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"bufio"
	"io"
	"archive/tar"
	"context"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"
)

type Manifest struct {
	Paths []string
}

func Load(ctx context.Context, path string) (m *Manifest, err error) {
	logger, ctx := logging.WithAttrs(ctx, "manifest", path)

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
	logger := logging.Get(ctx)

	t := tar.NewWriter(w)
	defer func() {
		err = t.Close()
	}()

	for _, p := range m.Paths {
		logger.Debug("adding path", "path", p)
	}

	return
}


func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")
	outputFlag := flag.String("output", common.Getenv("OUTPUT"), "write tarball to path")
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

	_, ctx = logging.WithAttrs(ctx, "tarball", output)
	err = m.CreateTarball(ctx, f)
	if err != nil {
		log.Fatal(err)
	}
}
