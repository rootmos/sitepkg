package manifest

import (
	"os"
	"path/filepath"
	"bufio"
	"io"
	"archive/tar"
	"context"
	"fmt"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"
)

type Manifest struct {
	Root string
	IgnoreMissing bool
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

func (m *Manifest) Has(path string) bool {
	for _, p := range m.Paths {
		if p == path {
			return true
		}
	}
	return false
}

func (m *Manifest) Add(path string) {
	if !m.Has(path) {
		m.Paths = append(m.Paths, path)
	}
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
	logger := logging.Get(ctx)
	wh := common.WriterSHA256(w)
	tw := tar.NewWriter(wh)
	defer func() {
		if e := tw.Close(); err == nil {
			err = e
		}
		if err == nil {
			logger.Debug("finished writing tarball", "SHA256", wh.HexDigest())
		}
	}()

	add := func(p string) (err error) {
		path := m.Resolve(ctx, p)
		logger, ctx = logging.WithAttrs(ctx, "name", p, "path", path)

		fi, err := os.Stat(path)
		if os.IsNotExist(err) && m.IgnoreMissing {
			logger.Info("ignoring missing")
			return nil
		}
		if err != nil {
			return err
		}

		logger, ctx = logging.WithAttrs(ctx, "mode", fi.Mode())

		hdr, err := tar.FileInfoHeader(fi, "")
		hdr.Name = p

		if err = tw.WriteHeader(hdr); err != nil {
			return err
		}

		if fi.IsDir() {
			logger.Info("add dir")
			return nil
		}

		if !fi.Mode().IsRegular() {
			// TODO: symlinks
			return fmt.Errorf("non-regular files not supported: %s", path)
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

		rh := common.ReaderSHA256(f)
		n, err := io.Copy(tw, rh)
		if err != nil {
			return err
		}
		logger.Info("add file", "bytes", n, "SHA256", rh.HexDigest())

		return
	}

	for _, p := range m.Paths {
		if err = add(p); err != nil {
			return
		}
	}

	return
}

func (m *Manifest) Extract(ctx context.Context, r io.Reader) error {
	logger := logging.Get(ctx)
	rh := common.ReaderSHA256(r)
	tr := tar.NewReader(rh)

	extract := func(hdr *tar.Header) (err error) {
		path := m.Resolve(ctx, hdr.Name)
		mode := os.FileMode(hdr.Mode)
		logger, _ := logging.WithAttrs(ctx, "name", hdr.Name, "path", path, "mode", mode)

		if hdr.Typeflag == tar.TypeDir {
			logger.Info("mkdir")
			return os.Mkdir(path, mode)
		}

		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("non-regular files not supported: %s", hdr.Name)
		}

		logger.Debug("opening")
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(hdr.Mode))
		if err != nil {
			return err
		}
		defer func() {
			if e := f.Close(); err == nil {
				err = e
			}
		}()

		logger.Debug("writing")
		rh := common.ReaderSHA256(tr)
		n, err := io.Copy(f, rh)
		if err != nil {
			return
		}
		logger.Info("extracted file", "bytes", n, "SHA256", rh.HexDigest())

		return
	}

	extracted := make(map[string]bool)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if !m.Has(hdr.Name) {
			logger.Debug("skipping", "name", hdr.Name)
			continue
		}

		if err := extract(hdr); err != nil {
			return err
		}

		extracted[hdr.Name] = true
	}

	logger.Debug("finished reading tarball", "SHA256", rh.HexDigest())

	for _, p := range m.Paths {
		if !extracted[p] {
			if m.IgnoreMissing {
				logger.Info("missing", "name", p)
			} else {
				return fmt.Errorf("not found in tarball: %s", p)
			}
		}
	}

	return nil
}
