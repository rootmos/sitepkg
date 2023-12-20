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
	Root string
	Paths []string
}

func Load(ctx context.Context, path, root string) (m *Manifest, err error) {
	logger, ctx := logging.WithAttrs(ctx, "manifest", path)

	f, err := os.Open(path)
	defer f.Close()

	m = &Manifest{
		Root: root,
	}

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
		if e := t.Close(); err == nil {
			err = e
		}
	}()

	add := func(p string) (err error) {
		q := p
		if filepath.IsLocal(p) {
			q = filepath.Join(m.Root, p)
			logger.Debug("resolved relative path", "rel", p, "abs", q)
		}

		logger, _ := logging.WithAttrs(ctx, "path", q)

		fi, err := os.Stat(q)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(fi, "")
		hdr.Name = p

		if err = t.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(q)
		if err != nil {
			return err
		}
		defer func() {
			if e := f.Close(); err == nil {
				err = e
			}
		}()

		n, err := io.Copy(t, f)
		if err != nil {
			return err
		}
		logger.Debug("wrote", "bytes", n)

		return nil
	}

	for _, p := range m.Paths {
		if err := add(p); err != nil {
			return nil
		}
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

	m, err := Load(ctx, *manifestFlag, root)
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
