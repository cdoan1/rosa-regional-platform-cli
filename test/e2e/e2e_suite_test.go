package e2e_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	binaryPath string
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

var _ = BeforeSuite(func() {
	ctx := context.Background()

	// CRITICAL: AWS_PROFILE must be set
	if os.Getenv("AWS_PROFILE") == "" {
		Fail("AWS_PROFILE environment variable must be set to run e2e tests.\n" +
			"Example: export AWS_PROFILE=your-profile-name")
	}

	// Determine binary path
	binaryPath = os.Getenv("E2E_BINARY_PATH")
	if binaryPath == "" {
		// Default to building in bin/ directory
		projectRoot, err := findProjectRoot()
		Expect(err).NotTo(HaveOccurred(), "Failed to find project root")

		binaryPath = filepath.Join(projectRoot, "bin", "rosactl")
	}

	// Check if binary exists, if not build it
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		By(fmt.Sprintf("Building rosactl binary at %s", binaryPath))
		err := buildBinary(binaryPath)
		Expect(err).NotTo(HaveOccurred(), "Failed to build rosactl binary")
	} else {
		By(fmt.Sprintf("Using existing binary at %s", binaryPath))
	}

	// Verify binary is executable
	err := exec.Command(binaryPath, "--version").Run()
	if err != nil {
		// If --version doesn't work, try --help
		err = exec.Command(binaryPath, "--help").Run()
	}
	Expect(err).NotTo(HaveOccurred(), "Binary is not executable or doesn't exist at "+binaryPath)

	// Optional: Clean up orphaned resources from previous failed test runs
	// This is helpful but not critical
	cleanupOrphanedResources(ctx)
})

var _ = AfterSuite(func() {
	// Cleanup is handled by individual tests
	// No global cleanup needed
})

// findProjectRoot finds the project root by looking for go.mod
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up directory tree looking for go.mod
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod not found)")
		}
		dir = parent
	}
}

// buildBinary builds the rosactl binary
func buildBinary(outputPath string) error {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return err
	}

	// Create bin directory if it doesn't exist
	binDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Build the binary
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/rosactl")
	cmd.Dir = projectRoot
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	return nil
}

// cleanupOrphanedResources attempts to cleanup resources from previous failed test runs
func cleanupOrphanedResources(ctx context.Context) {
	// This is a best-effort cleanup of resources that might have been left behind
	// We'll look for Lambda functions with the test prefix from over 1 hour ago

	defer GinkgoRecover()

	// Import the AWS helper here to avoid circular dependencies
	// We'll do this inline to keep it simple
	awsHelper, err := createAWSHelper(ctx)
	if err != nil {
		// If we can't create AWS helper, skip cleanup
		GinkgoWriter.Printf("Warning: Could not create AWS helper for orphaned resource cleanup: %v\n", err)
		return
	}

	// List all Lambda functions
	functions, err := awsHelper.ListFunctions(ctx)
	if err != nil {
		GinkgoWriter.Printf("Warning: Could not list functions for cleanup: %v\n", err)
		return
	}

	// Look for test functions created more than 1 hour ago
	cutoffTime := time.Now().Add(-1 * time.Hour).Unix()

	for _, fn := range functions {
		// Only cleanup functions with test prefix
		if len(fn) >= 5 && fn[:5] == "test-" {
			// Try to parse timestamp from function name (format: test-fn-TIMESTAMP)
			// This is a simple heuristic and won't catch all cases
			if shouldCleanupFunction(fn, cutoffTime) {
				GinkgoWriter.Printf("Cleaning up orphaned function: %s\n", fn)
				// Best effort cleanup - ignore errors
				_, _ = awsHelper.LambdaClient.DeleteFunction(ctx, &lambdaDeleteInput{
					FunctionName: &fn,
				})
			}
		}
	}
}

// shouldCleanupFunction determines if a function should be cleaned up
func shouldCleanupFunction(name string, cutoffTime int64) bool {
	// Simple heuristic: if name contains "test-" it might be a test function
	// We could parse timestamps from the name but keep it simple for now
	// Only cleanup if function name matches our test pattern
	return false // Disabled by default to be safe
}

// createAWSHelper creates an AWS helper for cleanup operations
func createAWSHelper(ctx context.Context) (*awsTestHelper, error) {
	// We'll implement a minimal version here
	// Import the actual helper when available
	return nil, fmt.Errorf("not implemented")
}

// Placeholder types for compilation
type awsTestHelper struct {
	LambdaClient interface {
		DeleteFunction(ctx context.Context, input *lambdaDeleteInput) (interface{}, error)
	}
}

func (h *awsTestHelper) ListFunctions(ctx context.Context) ([]string, error) {
	return nil, nil
}

type lambdaDeleteInput struct {
	FunctionName *string
}
