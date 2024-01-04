package main

import (
	"context"

	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/sealedbox"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func getKeyFromSMSecretValue(ctx context.Context, arn string) (*sealedbox.Key, error) {
	logger, ctx := logging.WithAttrs(ctx, "arn", arn)
	logger.Debug("fetching key")

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
	if err != nil {
		return nil, err
	}

	sm := secretsmanager.NewFromConfig(cfg)

	gsv, err := sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput {
		SecretId: aws.String(arn),
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		clear(gsv.SecretBinary)
		if gsv.SecretString != nil {
			clear([]byte(*gsv.SecretString))
		}
	}()

	logger.Debug("fetched secret value", "version", aws.ToString(gsv.VersionId))

	key, err := sealedbox.KeyFromBytes(gsv.SecretBinary)
	if err != nil {
		return nil, err
	}

	logger.Info("fetched key", "fpr", key.Fingerprint())

	return key, nil
}
