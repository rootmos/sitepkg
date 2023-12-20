package main

import (
	"log"
	"flag"
	"os"
	"path/filepath"

	"rootmos.io/sitepkg/internal/common"
)

func main() {
	chrootFlag := flag.String("chroot", common.Getenv("CHROOT"), "act relative directory")
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
}
