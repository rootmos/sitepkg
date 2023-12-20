package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"bufio"
	"io"
	"archive/tar"

	"rootmos.io/sitepkg/internal/common"
)

type Manifest struct {
	Paths []string
}

func Load(path string) (m *Manifest, err error) {
	f, err := os.Open(path)
	defer f.Close()

	m = &Manifest{}

	s := bufio.NewScanner(f)
	for s.Scan() {
		m.Paths = append(m.Paths, s.Text())
	}
	if err = s.Err(); err != nil {
		return
	}

	return m, nil
}

func (m *Manifest) CreateTarball(w io.Writer) (err error) {
	t := tar.NewWriter(w)
	defer func() {
		err = t.Close()
	}()

	return
}

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")
	outputFlag := flag.String("output", common.Getenv("OUTPUT"), "write tarball to path")
	flag.Parse()

	root := *chrootFlag
	var err error
	if root == "" {
		root, err = os.Getwd()
	} else {
		root, err = filepath.Abs(root)
	}
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("root: %v", root)

	if *manifestFlag == "" {
		log.Fatal("manifest not specified")
	}

	m, err := Load(*manifestFlag)
	if err != nil {
		log.Fatal(err)
	}

	log.Print(m)

	if *outputFlag == "" {
		log.Fatal("output not specified")
	}

	f, err := os.Create(*outputFlag)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = m.CreateTarball(f)
	if err != nil {
		log.Fatal(err)
	}
}
