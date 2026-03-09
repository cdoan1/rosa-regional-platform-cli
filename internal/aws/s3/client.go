package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awsconfig "github.com/openshift-online/rosa-regional-platform-cli/internal/aws"
)

type Client struct {
	s3  *s3.Client
	cfg aws.Config
}

func NewClient(ctx context.Context) (*Client, error) {
	cfg, err := awsconfig.NewConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &Client{
		s3:  s3.NewFromConfig(cfg),
		cfg: cfg,
	}, nil
}
