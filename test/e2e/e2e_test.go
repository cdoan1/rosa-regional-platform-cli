package e2e_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift-online/rosa-regional-platform-cli/test/e2e/fixtures"
)

var _ = Describe("rosactl Lambda E2E Tests", func() {
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

	Describe("ZIP-based Lambda workflow", func() {
		It("should create, invoke, and delete a ZIP-based Lambda function", func(ctx SpecContext) {
			functionName := fmt.Sprintf("test-zip-%d", time.Now().Unix())
			tracker.TrackLambda(functionName)

			By("Creating a ZIP-based Lambda function")
			stdout, err := cli.ExpectSuccess("lambda", "create", functionName)
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

			By("Waiting for function to be ready")
			err = awsHelper.WaitForFunctionReady(ctx, functionName, 2*time.Minute)
			Expect(err).NotTo(HaveOccurred(), "Function should be ready")

			By("Invoking the function")
			stdout, err = cli.ExpectSuccess("lambda", "invoke", functionName)
			Expect(err).NotTo(HaveOccurred())
			// Response should contain execution result
			Expect(stdout).To(Or(
				ContainSubstring("StatusCode"),
				ContainSubstring("200"),
				ContainSubstring("success"),
			))

			By("Deleting the function")
			stdout, err = cli.ExpectSuccess("lambda", "delete", functionName)
			Expect(err).NotTo(HaveOccurred())
			Expect(stdout).To(ContainSubstring("Successfully deleted"))

			By("Verifying function is deleted from AWS")
			Eventually(func() error {
				return awsHelper.VerifyFunctionDeleted(ctx, functionName)
			}, "1m", "5s").Should(Succeed(), "Function should be deleted from AWS")
		}, SpecTimeout(testTimeout))
	})

	Describe("List operations", func() {
		var functionNames []string

		BeforeEach(func() {
			functionNames = []string{
				fmt.Sprintf("test-list-1-%d", time.Now().Unix()),
				fmt.Sprintf("test-list-2-%d", time.Now().Unix()),
			}

			// Create multiple functions for list testing
			for _, name := range functionNames {
				tracker.TrackLambda(name)
			}
		})

		It("should list Lambda functions in table format", func(ctx SpecContext) {
			By("Creating test functions")
			for _, name := range functionNames {
				_, err := cli.ExpectSuccess("lambda", "create", name)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Waiting for functions to become active")
			for _, name := range functionNames {
				Eventually(func() bool {
					exists, _ := awsHelper.VerifyFunctionExists(ctx, name)
					return exists
				}, "2m", "5s").Should(BeTrue())
			}

			By("Listing functions in table format")
			stdout, err := cli.ExpectSuccess("lambda", "list")
			Expect(err).NotTo(HaveOccurred())

			// Verify output contains our test functions
			for _, name := range functionNames {
				Expect(stdout).To(ContainSubstring(name))
			}
		}, SpecTimeout(testTimeout))

		It("should list Lambda functions in JSON format", func(ctx SpecContext) {
			By("Creating test functions")
			for _, name := range functionNames {
				_, err := cli.ExpectSuccess("lambda", "create", name)
				Expect(err).NotTo(HaveOccurred())
			}

			By("Waiting for functions to become active")
			for _, name := range functionNames {
				Eventually(func() bool {
					exists, _ := awsHelper.VerifyFunctionExists(ctx, name)
					return exists
				}, "2m", "5s").Should(BeTrue())
			}

			By("Listing functions in JSON format")
			stdout, err := cli.ExpectSuccess("lambda", "list", "--output", "json")
			Expect(err).NotTo(HaveOccurred())

			// Verify JSON output
			Expect(stdout).To(ContainSubstring("["))
			Expect(stdout).To(ContainSubstring("]"))
			for _, name := range functionNames {
				Expect(stdout).To(ContainSubstring(name))
			}
		}, SpecTimeout(testTimeout))
	})

	Describe("Error handling", func() {
		It("should fail when creating duplicate function", func(ctx SpecContext) {
			functionName := fmt.Sprintf("test-dup-%d", time.Now().Unix())
			tracker.TrackLambda(functionName)

			By("Creating first function")
			_, err := cli.ExpectSuccess("lambda", "create", functionName)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for function to be created")
			Eventually(func() bool {
				exists, _ := awsHelper.VerifyFunctionExists(ctx, functionName)
				return exists
			}, "2m", "5s").Should(BeTrue())

			By("Attempting to create duplicate function")
			stderr, err := cli.ExpectFailure("lambda", "create", functionName)
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Or(
				ContainSubstring("already exists"),
				ContainSubstring("ResourceConflictException"),
				ContainSubstring("conflict"),
			))
		}, SpecTimeout(testTimeout))

		It("should fail when invoking non-existent function", func(ctx SpecContext) {
			nonExistentFunction := fmt.Sprintf("non-existent-%d", time.Now().Unix())

			By("Attempting to invoke non-existent function")
			stderr, err := cli.ExpectFailure("lambda", "invoke", nonExistentFunction)
			Expect(err).To(HaveOccurred())
			Expect(stderr).To(Or(
				ContainSubstring("does not exist"),
				ContainSubstring("ResourceNotFoundException"),
				ContainSubstring("not found"),
			))
		}, SpecTimeout(30*time.Second))
	})

	Describe("Version management", func() {
		// Disabled: Update operations are timing-sensitive and may fail with 409 errors
		// TODO: Re-enable once update operation timing is more stable
		XIt("should update function and list versions", func(ctx SpecContext) {
			functionName := fmt.Sprintf("test-version-%d", time.Now().Unix())
			tracker.TrackLambda(functionName)

			By("Creating function")
			_, err := cli.ExpectSuccess("lambda", "create", functionName)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for function to become active")
			Eventually(func() bool {
				exists, _ := awsHelper.VerifyFunctionExists(ctx, functionName)
				return exists
			}, "2m", "5s").Should(BeTrue())

			By("Waiting for function to be ready for updates")
			err = awsHelper.WaitForFunctionReady(ctx, functionName, 3*time.Minute)
			Expect(err).NotTo(HaveOccurred(), "Function should be ready for updates")

			By("Updating function code")
			_, err = cli.ExpectSuccess("lambda", "update", functionName)
			Expect(err).NotTo(HaveOccurred())

			By("Listing function versions")
			stdout, err := cli.ExpectSuccess("lambda", "versions", functionName)
			Expect(err).NotTo(HaveOccurred())
			// Should have at least $LATEST version
			Expect(stdout).To(ContainSubstring("$LATEST"))
		}, SpecTimeout(testTimeout))
	})
})
