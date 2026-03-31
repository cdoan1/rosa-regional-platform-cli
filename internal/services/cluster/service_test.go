package cluster

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"VpcId", "vpcId"},
		{"SubnetIds", "subnetIds"},
		{"OIDCProviderArn", "oidcProviderArn"},
		{"OIDCProviderURL", "oidcProviderURL"},
		{"WorkerRoleArn", "workerRoleArn"},
		{"WorkerInstanceProfileName", "workerInstanceProfileName"},
		{"ControlPlaneRoleArn", "controlPlaneRoleArn"},
		{"", ""},
		{"A", "a"},
		{"ABC", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
