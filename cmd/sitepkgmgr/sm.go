package main

import (
	"context"
	"fmt"

	"rootmos.io/sitepkg/internal/logging"
	"rootmos.io/sitepkg/sealedbox"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

var smClient *secretsmanager.Client

func getSM(ctx context.Context) (*secretsmanager.Client, error) {
	if smClient != nil {
		return smClient, nil
	}

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithEC2IMDSRegion(),
	)
	if err != nil {
		return nil, err
	}

	smClient = secretsmanager.NewFromConfig(cfg)
	return smClient, nil
}

func doNewSMSecretValue(ctx context.Context, arn string, force bool) error {
	logger, ctx := logging.WithAttrs(ctx, "arn", arn)
	logger.Debug("populating secretsmanager secret value")

	sm, err := getSM(ctx)
	if err != nil {
		return err
	}

	ds, err := sm.DescribeSecret(ctx, &secretsmanager.DescribeSecretInput {
		SecretId: aws.String(arn),
	})

	if ds == nil {
		logger.Debug("secret not found")
		return fmt.Errorf("secret not found: %s", arn)
	}

	var current string
	for k, stages := range ds.VersionIdsToStages {
		for _, stage := range stages {
			if stage == "AWSCURRENT" {
				current = k
				break
			}
		}
		if current != "" {
			break
		}
	}

	if current == "" {
		logger.Debug("no secret versions")
	} else {
		if force {
			logger.Warn("overwriting current secret value", "version", current)
		} else {
			logger.Debug("refusing to overwrite current secret value", "version", current)
			return fmt.Errorf("refusing to overwrite current secret value; %s: %s", arn, current)
		}
	}

	key, err := sealedbox.NewKey()
	if err != nil {
		return err
	}
	defer key.Close()

	logger = logger.With("fpr", key.Fingerprint())
	logger.Info("generated new key")

	psv, err := sm.PutSecretValue(ctx, &secretsmanager.PutSecretValueInput {
		SecretId: aws.String(arn),
		SecretBinary: key.Bytes(),
	})
	if err != nil {
		return err
	}

	logger.Info("created new secret value", "version", aws.ToString(psv.VersionId))

	return nil
}
