package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// CreatePublicBucket creates an S3 bucket with public read access for OIDC discovery documents
func (c *Client) CreatePublicBucket(ctx context.Context, bucketName string) error {
	region := c.cfg.Region

	// Create bucket
	createInput := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}

	// Only set LocationConstraint for regions other than us-east-1
	if region != "us-east-1" {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := c.s3.CreateBucket(ctx, createInput)
	if err != nil {
		var bucketExists *types.BucketAlreadyExists
		var bucketOwnedByYou *types.BucketAlreadyOwnedByYou
		if !errors.As(err, &bucketExists) && !errors.As(err, &bucketOwnedByYou) {
			return &S3Error{
				Operation: "create",
				Message:   fmt.Sprintf("failed to create bucket: %v", err),
			}
		}
	}

	// Configure public access block settings
	_, err = c.s3.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(false),
			IgnorePublicAcls:      aws.Bool(false),
			BlockPublicPolicy:     aws.Bool(false),
			RestrictPublicBuckets: aws.Bool(false),
		},
	})
	if err != nil {
		return &S3Error{
			Operation: "configure public access",
			Message:   fmt.Sprintf("failed to configure public access: %v", err),
		}
	}

	// Set bucket policy for public read
	policy := fmt.Sprintf(`{
  "Version": "2012-10-17",
  "Statement": [{
    "Sid": "PublicReadGetObject",
    "Effect": "Allow",
    "Principal": "*",
    "Action": "s3:GetObject",
    "Resource": "arn:aws:s3:::%s/*"
  }]
}`, bucketName)

	_, err = c.s3.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(bucketName),
		Policy: aws.String(policy),
	})
	if err != nil {
		return &S3Error{
			Operation: "set bucket policy",
			Message:   fmt.Sprintf("failed to set bucket policy: %v", err),
		}
	}

	return nil
}

// UploadObject uploads content to an S3 bucket
func (c *Client) UploadObject(ctx context.Context, bucketName, key, content, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        strings.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return &S3Error{
			Operation: "upload",
			Message:   fmt.Sprintf("failed to upload object: %v", err),
		}
	}
	return nil
}

// BucketInfo represents information about an S3 bucket
type BucketInfo struct {
	Name         string
	CreationDate string
	Region       string
}

// ListBucketsWithPrefix lists all S3 buckets with a specific prefix
func (c *Client) ListBucketsWithPrefix(ctx context.Context, prefix string) ([]BucketInfo, error) {
	output, err := c.s3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, &S3Error{
			Operation: "list",
			Message:   fmt.Sprintf("failed to list buckets: %v", err),
		}
	}

	var buckets []BucketInfo
	for _, bucket := range output.Buckets {
		bucketName := aws.ToString(bucket.Name)
		if strings.HasPrefix(bucketName, prefix) {
			// Get bucket region
			location, err := c.s3.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
				Bucket: aws.String(bucketName),
			})
			region := "us-east-1" // Default
			if err == nil && location.LocationConstraint != "" {
				region = string(location.LocationConstraint)
			}

			buckets = append(buckets, BucketInfo{
				Name:         bucketName,
				CreationDate: aws.ToTime(bucket.CreationDate).Format("2006-01-02 15:04:05"),
				Region:       region,
			})
		}
	}

	return buckets, nil
}

// DeleteBucket deletes an S3 bucket and all its contents
func (c *Client) DeleteBucket(ctx context.Context, bucketName string) error {
	// First, delete all objects in the bucket
	listOutput, err := c.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return &S3Error{
			Operation: "delete",
			Message:   fmt.Sprintf("failed to list objects: %v", err),
		}
	}

	// Delete all objects
	for _, obj := range listOutput.Contents {
		_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    obj.Key,
		})
		if err != nil {
			return &S3Error{
				Operation: "delete",
				Message:   fmt.Sprintf("failed to delete object %s: %v", aws.ToString(obj.Key), err),
			}
		}
	}

	// Delete the bucket
	_, err = c.s3.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return &S3Error{
			Operation: "delete",
			Message:   fmt.Sprintf("failed to delete bucket: %v", err),
		}
	}

	return nil
}

// BucketExists checks if a bucket exists
func (c *Client) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	_, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		return false, &S3Error{
			Operation: "check existence",
			Message:   fmt.Sprintf("failed to check bucket existence: %v", err),
		}
	}
	return true, nil
}
