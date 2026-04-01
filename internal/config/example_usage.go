package config

// Example usage for other commands that need to call the platform API:
//
// import (
//     "fmt"
//     "net/http"
//     "github.com/openshift-online/rosa-regional-platform-cli/internal/config"
// )
//
// func callPlatformAPI() error {
//     // Get the platform API URL
//     baseURL, err := config.GetPlatformAPIURL()
//     if err != nil {
//         return err
//     }
//
//     // Build the full API endpoint
//     endpoint := fmt.Sprintf("%s/api/v1/clusters", baseURL)
//
//     // Make the API call
//     resp, err := http.Get(endpoint)
//     if err != nil {
//         return err
//     }
//     defer resp.Body.Close()
//
//     // Process response...
//     return nil
// }
//
// Or use MustGetPlatformAPIURL when the URL is required:
//
// func createCluster() error {
//     baseURL := config.MustGetPlatformAPIURL()  // Will exit if not configured
//     endpoint := fmt.Sprintf("%s/api/v1/clusters", baseURL)
//     // ... rest of implementation
//     return nil
// }
