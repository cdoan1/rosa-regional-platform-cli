package localstack_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLocalStack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LocalStack Integration Suite")
}
