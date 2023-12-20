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

func (m *Manifest) Resolve(ctx context.Context, p string) string {
	logger := logging.Get(ctx)

	q := p
	if filepath.IsLocal(p) {
		q = filepath.Join(m.Root, p)
		logger.Debug("resolved relative path", "rel", p, "abs", q)
	}

	return q
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

func (m *Manifest) Create(ctx context.Context, w io.Writer) (err error) {
	tw := tar.NewWriter(w)
	defer func() {
		if e := tw.Close(); err == nil {
			err = e
		}
	}()

	add := func(p string) (err error) {
		path := m.Resolve(ctx, p)
		logger, _ := logging.WithAttrs(ctx, "path", path)
		logger.Info("adding", "name", p)

		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(fi, "") // TODO: symlinks
		hdr.Name = p

		if err = tw.WriteHeader(hdr); err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			if e := f.Close(); err == nil {
				err = e
			}
		}()

		n, err := io.Copy(tw, f)
		if err != nil {
			return err
		}
		logger.Debug("wrote", "bytes", n)

		return
	}

	for _, p := range m.Paths {
		if err = add(p); err != nil {
			return
		}
	}

	return
}

func (m *Manifest) Extract(ctx context.Context, r io.Reader) (err error) {
	tr := tar.NewReader(r)

	extract := func(hdr *tar.Header) (err error) {
		path := m.Resolve(ctx, hdr.Name)
		logger, _ := logging.WithAttrs(ctx, "path", path)
		logger.Info("extracting", "name", hdr.Name)

		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		defer func() {
			if e := f.Close(); err == nil {
				err = e
			}
		}()

		n, err := io.Copy(f, tr)
		if err != nil {
			return
		}
		logger.Debug("wrote", "bytes", n)

		return
	}

	for {
		var hdr *tar.Header
		hdr, err = tr.Next()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}

		if err = extract(hdr); err != nil {
			return
		}
	}

	return
}

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")

	createFlag := flag.String("create", common.Getenv("CREATE"), "write tarball")
	extractFlag := flag.String("extract", common.Getenv("EXTRACT"), "extract tarball")
	// verifyFlag := flag.String("verify", common.Getenv("VERIFY"), "verify tarball") // or status? check? test?

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

	const (
		Noop = iota
		Create
		Extract
	)
	action := Noop

	var tarball string
	if *createFlag != "" {
		if action != Noop {
			log.Fatal("more than one action specified")
		}
		action = Create
		tarball = *createFlag
	}
	if *extractFlag != "" {
		if action != Noop {
			log.Fatal("more than one action specified")
		}
		action = Extract
		tarball = *extractFlag
	}

	logger, ctx = logging.WithAttrs(ctx, "tarball", tarball)

	switch action {
	case Create:
		logger.Info("creating")
		f, err := os.Create(tarball)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if err := m.Create(ctx, f); err != nil {
			log.Fatal(err)
		}
	case Extract:
		logger.Info("extracting")
		f, err := os.Open(tarball)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		if err := m.Extract(ctx, f); err != nil {
			log.Fatal(err)
		}
	}
}
