package oidc

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	awsconfig "github.com/openshift-online/rosa-regional-platform-cli/internal/aws"
)

type Client struct {
	iam *iam.Client
	cfg aws.Config
}

func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := awsconfig.NewConfig(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		iam: iam.NewFromConfig(cfg),
		cfg: cfg,
	}, nil
}
