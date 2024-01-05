package main

import (
	"log"
	"flag"
	"context"

	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/internal/common"
	"rootmos.io/sitepkg/sealedbox"
)

func doNewKeyfile(ctx context.Context, path string, force bool) error {
	logger, ctx := logging.WithAttrs(ctx, "path", path)

	logger.Debug("creating new keyfile")
	key, err := sealedbox.NewKeyfile(path, force)
	if err != nil {
		return err
	}
	defer key.Close()

	logger.Info("created new keyfile", "path", path, "fpr", key.Fingerprint())
	return nil
}

func main() {
	newKeyfile := flag.String("new-keyfile", common.Getenv("NEW_KEYFILE"), "create new keyfile")
	newAwsSecretsManagerSecretValue := flag.String("new-aws-secretsmanager-secret-value", common.Getenv("NEW_AWS_SECRETSMANAGER_SECRET_VALUE_ARN"), "populate the secret value of the AWS Secrets Manager Secret specified by its ARN")
	force := flag.Bool("force", common.GetenvBool("FORCE"), "overwrite key if exists")
	logConfig := logging.PrepareConfig(common.EnvPrefix)
	flag.Parse()

	logger, err := logConfig.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	logger.Debug("hello")

	ctx := logging.Set(context.Background(), logger)

	if *newKeyfile != "" {
		if err := doNewKeyfile(ctx, *newKeyfile, *force); err != nil {
			log.Fatal(err)
		}
	}

	if *newAwsSecretsManagerSecretValue != "" {
		if err := doNewSMSecretValue(ctx, *newAwsSecretsManagerSecretValue, *force); err != nil {
			log.Fatal(err)
		}
	}
}
