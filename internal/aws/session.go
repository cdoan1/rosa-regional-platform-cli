package aws

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

func NewConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*config.LoadOptions) error

	if region := os.Getenv("ROSACTL_REGION"); region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	if profile := os.Getenv("ROSACTL_PROFILE"); profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}
