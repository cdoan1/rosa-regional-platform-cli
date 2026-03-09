package e2e_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/rosa-regional-platform-cli/test/e2e/fixtures"
)

var _ = Describe("rosactl OIDC E2E Tests", func() {
	var (
		cli         *fixtures.CLIRunner
		awsHelper   *fixtures.AWSTestHelper
		tracker     *fixtures.ResourceTracker
		ctx         context.Context
		testTimeout time.Duration
	)

	BeforeEach(func() {
		ctx = context.Background()
		testTimeout = 5 * time.Minute

		cli = fixtures.NewCLIRunner(binaryPath)

		var err error
		awsHelper, err = fixtures.NewAWSTestHelper(ctx)
		Expect(err).NotTo(HaveOccurred(), "Failed to create AWS helper")

		tracker = fixtures.NewResourceTracker()
	})

	AfterEach(func() {
		// Always cleanup resources, even if test fails
		By("Cleaning up test resources")
		err := tracker.CleanupAll(ctx, awsHelper)
		if err != nil {
			GinkgoWriter.Printf("Warning: Cleanup failed: %v\n", err)
		}
	})

	Describe("OIDC Lambda creation", func() {
		It("should create OIDC Lambda function with embedded RSA keys", func(ctx SpecContext) {
			functionName := fmt.Sprintf("test-oidc-%d", time.Now().Unix())
			tracker.TrackLambda(functionName)

			By("Creating OIDC Lambda function")
			stdout, err := cli.ExpectSuccess("lambda", "create", functionName, "--handler", "oidc")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Generating RSA key pair"))
			Expect(stdout).To(ContainSubstring("Generated RSA key pair"))
			Expect(stdout).To(ContainSubstring("Successfully created"))

			By("Verifying function exists in AWS")
			Eventually(func() bool {
				exists, err := awsHelper.VerifyFunctionExists(ctx, functionName)
				if err != nil {
					GinkgoWriter.Printf("Error checking function: %v\n", err)
					return false
				}
				return exists
			}, "2m", "5s").Should(BeTrue(), "Function should exist in AWS")

			By("Verifying function has RSA key environment variables")
			env, err := awsHelper.GetFunctionEnvironment(ctx, functionName)
			Expect(err).NotTo(HaveOccurred())
			Expect(env).To(HaveKey("JWK_N"), "Should have JWK_N environment variable")
			Expect(env).To(HaveKey("JWK_E"), "Should have JWK_E environment variable")
			Expect(env).To(HaveKey("JWK_KID"), "Should have JWK_KID environment variable")
		}, SpecTimeout(testTimeout))

		It("should create OIDC delete Lambda function", func(ctx SpecContext) {
			functionName := fmt.Sprintf("test-oidc-delete-%d", time.Now().Unix())
			tracker.TrackLambda(functionName)

			By("Creating OIDC delete Lambda function")
			stdout, err := cli.ExpectSuccess("lambda", "create", functionName, "--handler", "oidc-delete")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Successfully created"))

			By("Verifying function exists in AWS")
			Eventually(func() bool {
				exists, err := awsHelper.VerifyFunctionExists(ctx, functionName)
				if err != nil {
					GinkgoWriter.Printf("Error checking function: %v\n", err)
					return false
				}
				return exists
			}, "2m", "5s").Should(BeTrue(), "Function should exist in AWS")
		}, SpecTimeout(testTimeout))
	})

	Describe("OIDC issuer lifecycle", func() {
		var oidcLambdaName string
		var oidcDeleteLambdaName string

		BeforeEach(func() {
			timestamp := time.Now().Unix()
			oidcLambdaName = fmt.Sprintf("test-oidc-%d", timestamp)
			oidcDeleteLambdaName = fmt.Sprintf("test-oidc-delete-%d", timestamp)

			// Create OIDC Lambda functions before each test
			By("Setting up OIDC Lambda functions")
			tracker.TrackLambda(oidcLambdaName)
			tracker.TrackLambda(oidcDeleteLambdaName)

			_, err := cli.ExpectSuccess("lambda", "create", oidcLambdaName, "--handler", "oidc")
			Expect(err).NotTo(HaveOccurred())

			_, err = cli.ExpectSuccess("lambda", "create", oidcDeleteLambdaName, "--handler", "oidc-delete")
			Expect(err).NotTo(HaveOccurred())

			// Wait for both functions to be ready
			Eventually(func() bool {
				exists, _ := awsHelper.VerifyFunctionExists(ctx, oidcLambdaName)
				return exists
			}, "2m", "5s").Should(BeTrue())

			Eventually(func() bool {
				exists, _ := awsHelper.VerifyFunctionExists(ctx, oidcDeleteLambdaName)
				return exists
			}, "2m", "5s").Should(BeTrue())

			err = awsHelper.WaitForFunctionReady(ctx, oidcLambdaName, 2*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			err = awsHelper.WaitForFunctionReady(ctx, oidcDeleteLambdaName, 2*time.Minute)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create, list, and delete an OIDC issuer", func(ctx SpecContext) {
			issuerName := fmt.Sprintf("e2e-test-%d", time.Now().Unix())
			bucketName := fmt.Sprintf("oidc-issuer-%s", issuerName)

			tracker.TrackS3Bucket(bucketName)

			By("Creating OIDC issuer")
			stdout, err := cli.ExpectSuccess("oidc", "create", issuerName, "--region", awsHelper.Region, "--function", oidcLambdaName)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Using bucket name"))
			Expect(stdout).To(ContainSubstring(bucketName))
			Expect(stdout).To(ContainSubstring("OIDC issuer created successfully"))

			// Extract provider ARN from output for cleanup
			if strings.Contains(stdout, "provider_arn") {
				// Try to extract ARN from JSON output
				lines := strings.Split(stdout, "\n")
				for _, line := range lines {
					if strings.Contains(line, "provider_arn") && strings.Contains(line, "arn:aws:iam") {
						// This is a best-effort extraction
						start := strings.Index(line, "arn:aws:iam")
						if start >= 0 {
							end := strings.Index(line[start:], "\"")
							if end > 0 {
								arn := line[start : start+end]
								tracker.TrackOIDCProvider(arn)
							}
						}
					}
				}
			}

			By("Verifying S3 bucket exists")
			exists, err := awsHelper.VerifyS3BucketExists(ctx, bucketName)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue(), "S3 bucket should exist")

			By("Verifying OIDC discovery documents exist")
			hasDiscovery, err := awsHelper.VerifyOIDCDiscoveryExists(ctx, bucketName)
			Expect(err).NotTo(HaveOccurred())
			Expect(hasDiscovery).To(BeTrue(), "OIDC discovery documents should exist")

			By("Verifying IAM OIDC provider exists")
			issuerURL := fmt.Sprintf("%s.s3.%s.amazonaws.com", bucketName, awsHelper.Region)
			providerExists, err := awsHelper.VerifyOIDCProviderExists(ctx, issuerURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(providerExists).To(BeTrue(), "IAM OIDC provider should exist")

			By("Listing OIDC issuers")
			stdout, err = cli.ExpectSuccess("oidc", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring(bucketName))
			Expect(stdout).To(ContainSubstring("Active"))

			By("Deleting OIDC issuer")
			stdout, err = cli.ExpectSuccess("oidc", "delete", issuerName, "--region", awsHelper.Region, "--function", oidcDeleteLambdaName)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("OIDC issuer deleted successfully"))

			By("Verifying S3 bucket is deleted")
			Eventually(func() bool {
				exists, err := awsHelper.VerifyS3BucketExists(ctx, bucketName)
				if err != nil {
					GinkgoWriter.Printf("Error checking bucket %s: %v\n", bucketName, err)
					return false
				}
				if exists {
					GinkgoWriter.Printf("Bucket %s still exists, waiting...\n", bucketName)
				}
				return !exists
			}, "5m", "10s").Should(BeTrue(), "S3 bucket should be deleted")

			By("Verifying IAM OIDC provider is deleted")
			Eventually(func() bool {
				exists, _ := awsHelper.VerifyOIDCProviderExists(ctx, issuerURL)
				return !exists
			}, "5m", "10s").Should(BeTrue(), "IAM OIDC provider should be deleted")
		}, SpecTimeout(testTimeout))

		It("should list OIDC issuers in JSON format", func(ctx SpecContext) {
			issuerName := fmt.Sprintf("e2e-json-%d", time.Now().Unix())
			bucketName := fmt.Sprintf("oidc-issuer-%s", issuerName)

			tracker.TrackS3Bucket(bucketName)

			By("Creating OIDC issuer")
			_, err := cli.ExpectSuccess("oidc", "create", issuerName, "--region", awsHelper.Region, "--function", oidcLambdaName)
			Expect(err).NotTo(HaveOccurred())

			By("Listing OIDC issuers in JSON format")
			stdout, err := cli.ExpectSuccess("oidc", "list", "--output", "json")
			Expect(err).NotTo(HaveOccurred())

			// Verify JSON output
			Expect(stdout).To(ContainSubstring("["))
			Expect(stdout).To(ContainSubstring("]"))
			Expect(stdout).To(ContainSubstring(bucketName))
			Expect(stdout).To(Or(
				ContainSubstring("\"status\""),
				ContainSubstring("Active"),
			))

			By("Cleaning up OIDC issuer")
			_, _, _ = cli.Run("oidc", "delete", issuerName, "--region", awsHelper.Region, "--function", oidcDeleteLambdaName)
		}, SpecTimeout(testTimeout))

		It("should handle multiple OIDC issuers", func(ctx SpecContext) {
			issuer1 := fmt.Sprintf("e2e-multi1-%d", time.Now().Unix())
			issuer2 := fmt.Sprintf("e2e-multi2-%d", time.Now().Unix())
			bucket1 := fmt.Sprintf("oidc-issuer-%s", issuer1)
			bucket2 := fmt.Sprintf("oidc-issuer-%s", issuer2)

			tracker.TrackS3Bucket(bucket1)
			tracker.TrackS3Bucket(bucket2)

			By("Creating first OIDC issuer")
			_, err := cli.ExpectSuccess("oidc", "create", issuer1, "--region", awsHelper.Region, "--function", oidcLambdaName)
			Expect(err).NotTo(HaveOccurred())

			By("Creating second OIDC issuer")
			_, err = cli.ExpectSuccess("oidc", "create", issuer2, "--region", awsHelper.Region, "--function", oidcLambdaName)
			Expect(err).NotTo(HaveOccurred())

			By("Listing OIDC issuers")
			stdout, err := cli.ExpectSuccess("oidc", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring(bucket1))
			Expect(stdout).To(ContainSubstring(bucket2))

			By("Deleting first OIDC issuer")
			_, err = cli.ExpectSuccess("oidc", "delete", issuer1, "--region", awsHelper.Region, "--function", oidcDeleteLambdaName)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying first issuer is deleted but second remains")
			stdout, err = cli.ExpectSuccess("oidc", "list")
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).NotTo(ContainSubstring(bucket1))
			Expect(stdout).To(ContainSubstring(bucket2))

			By("Deleting second OIDC issuer")
			_, err = cli.ExpectSuccess("oidc", "delete", issuer2, "--region", awsHelper.Region, "--function", oidcDeleteLambdaName)
			Expect(err).NotTo(HaveOccurred())
		}, SpecTimeout(testTimeout))
	})

	Describe("OIDC error handling", func() {
		It("should fail when OIDC Lambda is not created", func(ctx SpecContext) {
			issuerName := fmt.Sprintf("e2e-nolambda-%d", time.Now().Unix())

			By("Attempting to create OIDC issuer without Lambda function")
			stderr, err := cli.ExpectFailure("oidc", "create", issuerName)
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Or(
				ContainSubstring("not found"),
				ContainSubstring("does not exist"),
			))
		}, SpecTimeout(30*time.Second))

		It("should show auto-prefixing in output", func(ctx SpecContext) {
			// Create OIDC Lambda first
			oidcLambda := fmt.Sprintf("test-oidc-%d", time.Now().Unix())
			oidcDeleteLambda := fmt.Sprintf("test-oidc-delete-%d", time.Now().Unix())

			tracker.TrackLambda(oidcLambda)
			tracker.TrackLambda(oidcDeleteLambda)

			_, err := cli.ExpectSuccess("lambda", "create", oidcLambda, "--handler", "oidc")
			Expect(err).NotTo(HaveOccurred())

			_, err = cli.ExpectSuccess("lambda", "create", oidcDeleteLambda, "--handler", "oidc-delete")
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				exists, _ := awsHelper.VerifyFunctionExists(ctx, oidcLambda)
				return exists
			}, "2m", "5s").Should(BeTrue())

			err = awsHelper.WaitForFunctionReady(ctx, oidcLambda, 2*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			err = awsHelper.WaitForFunctionReady(ctx, oidcDeleteLambda, 2*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			issuerName := "short-name"
			expectedBucket := "oidc-issuer-short-name"

			tracker.TrackS3Bucket(expectedBucket)

			By("Creating OIDC issuer with short name")
			stdout, err := cli.ExpectSuccess("oidc", "create", issuerName, "--region", awsHelper.Region, "--function", oidcLambda)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Using bucket name"))
			Expect(stdout).To(ContainSubstring(expectedBucket))

			By("Cleaning up")
			_, _, _ = cli.Run("oidc", "delete", issuerName, "--region", awsHelper.Region, "--function", oidcDeleteLambda)
		}, SpecTimeout(testTimeout))
	})
})
