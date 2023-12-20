package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"
	"context"

	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/internal/manifest"
)

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
	m, err := manifest.Load(ctx, *manifestFlag, root)
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
