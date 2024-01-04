package main

import (
	"log"
	"flag"

	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/sealedbox"
)

func main() {
	newKeyfile := flag.String("new-keyfile", common.Getenv("NEW_KEYFILE"), "create new keyfile")
	flag.Parse()

	logger, err := logging.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug("hello")

	if *newKeyfile != "" {
		path := *newKeyfile
		logger.Debug("creating new keyfile", "path", path)
		key, err := sealedbox.NewKeyfile(path)
		if err != nil {
			log.Fatal(err)
		}
		key.Close()
		logger.Info("created new keyfile", "path", path)
	}
}
