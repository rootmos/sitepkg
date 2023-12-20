package manifest

import (
	"os"
	"path/filepath"
	"bufio"
	"io"
	"archive/tar"
	"context"

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
