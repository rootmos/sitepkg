package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"bufio"

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

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
	manifestFlag := flag.String("manifest", common.Getenv("MANIFEST"), "manifest path")
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
}
