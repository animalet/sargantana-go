package controller

import (
	"os"
	"testing"

	"github.com/markbates/goth"
)

func TestMockServerIntegration(t *testing.T) {
	// Save original environment
	originalMockURL := os.Getenv("OAUTH_MOCK_SERVER_URL")
	originalGoogleKey := os.Getenv("GOOGLE_KEY")
	originalGoogleSecret := os.Getenv("GOOGLE_SECRET")
	originalGitHubKey := os.Getenv("GITHUB_KEY")
	originalGitHubSecret := os.Getenv("GITHUB_SECRET")
	originalFacebookKey := os.Getenv("FACEBOOK_KEY")
	originalFacebookSecret := os.Getenv("FACEBOOK_SECRET")
	originalTwitterKey := os.Getenv("TWITTER_KEY")
	originalTwitterSecret := os.Getenv("TWITTER_SECRET")

	// Clean up after test
	defer func() {
		_ = os.Setenv("OAUTH_MOCK_SERVER_URL", originalMockURL)
		_ = os.Setenv("GOOGLE_KEY", originalGoogleKey)
		_ = os.Setenv("GOOGLE_SECRET", originalGoogleSecret)
		_ = os.Setenv("GITHUB_KEY", originalGitHubKey)
		_ = os.Setenv("GITHUB_SECRET", originalGitHubSecret)
		_ = os.Setenv("FACEBOOK_KEY", originalFacebookKey)
		_ = os.Setenv("FACEBOOK_SECRET", originalFacebookSecret)
		_ = os.Setenv("TWITTER_KEY", originalTwitterKey)
		_ = os.Setenv("TWITTER_SECRET", originalTwitterSecret)
		goth.ClearProviders()
	}()

	tests := []struct {
		name          string
		mockServerURL string
		googleKey     string
		githubKey     string
		facebookKey   string
		twitterKey    string
		expectedCount int
		shouldUseMock bool
	}{
		{
			name:          "no mock server configured",
			mockServerURL: "",
			googleKey:     "",
			githubKey:     "",
			facebookKey:   "",
			twitterKey:    "",
			expectedCount: 0,
			shouldUseMock: false,
		},
		{
			name:          "mock server with google provider only",
			mockServerURL: "http://localhost:8080",
			googleKey:     "mock-google-key",
			githubKey:     "",
			facebookKey:   "",
			twitterKey:    "",
			expectedCount: 1,
			shouldUseMock: true,
		},
		{
			name:          "mock server with all four providers",
			mockServerURL: "http://localhost:8080",
			googleKey:     "mock-google-key",
			githubKey:     "mock-github-key",
			facebookKey:   "mock-facebook-key",
			twitterKey:    "mock-twitter-key",
			expectedCount: 4,
			shouldUseMock: true,
		},
		{
			name:          "mock server with facebook and twitter only",
			mockServerURL: "http://localhost:8080",
			googleKey:     "",
			githubKey:     "",
			facebookKey:   "mock-facebook-key",
			twitterKey:    "mock-twitter-key",
			expectedCount: 2,
			shouldUseMock: true,
		},
		{
			name:          "mock server configured but no keys",
			mockServerURL: "http://localhost:8080",
			googleKey:     "",
			githubKey:     "",
			facebookKey:   "",
			twitterKey:    "",
			expectedCount: 0,
			shouldUseMock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear providers before each test
			goth.ClearProviders()

			// Set up environment
			_ = os.Setenv("OAUTH_MOCK_SERVER_URL", tt.mockServerURL)
			_ = os.Setenv("GOOGLE_KEY", tt.googleKey)
			_ = os.Setenv("GOOGLE_SECRET", "mock-secret")
			_ = os.Setenv("GITHUB_KEY", tt.githubKey)
			_ = os.Setenv("GITHUB_SECRET", "mock-secret")
			_ = os.Setenv("FACEBOOK_KEY", tt.facebookKey)
			_ = os.Setenv("FACEBOOK_SECRET", "mock-secret")
			_ = os.Setenv("TWITTER_KEY", tt.twitterKey)
			_ = os.Setenv("TWITTER_SECRET", "mock-secret")

			// Test isMockMode function
			if isMockMode() != tt.shouldUseMock {
				t.Errorf("isMockMode() = %v, want %v", isMockMode(), tt.shouldUseMock)
			}

			// Test setupMockProviders function
			providers := setupMockProviders("http://localhost:3000/auth/%s/callback")

			if len(providers) != tt.expectedCount {
				t.Errorf("setupMockProviders() returned %d providers, want %d", len(providers), tt.expectedCount)
			}

			// Verify provider configuration when mock is enabled
			if tt.shouldUseMock && len(providers) > 0 {
				expectedProviders := make(map[string]bool)
				if tt.googleKey != "" {
					expectedProviders["google"] = true
				}
				if tt.githubKey != "" {
					expectedProviders["github"] = true
				}
				if tt.facebookKey != "" {
					expectedProviders["facebook"] = true
				}
				if tt.twitterKey != "" {
					expectedProviders["twitter"] = true
				}

				for _, provider := range providers {
					// Verify the provider is properly configured
					if provider.Name() == "" {
						t.Error("Provider name should not be empty")
					}

					// Verify this is an expected provider
					if !expectedProviders[provider.Name()] {
						t.Errorf("Unexpected provider: %s", provider.Name())
					}
					delete(expectedProviders, provider.Name())
				}

				// Verify all expected providers were found
				for providerName := range expectedProviders {
					t.Errorf("Expected provider %s not found", providerName)
				}
			}
		})
	}
}

func TestCreateMockProviders(t *testing.T) {
	// Save original environment
	originalMockURL := os.Getenv("OAUTH_MOCK_SERVER_URL")
	originalGoogleKey := os.Getenv("GOOGLE_KEY")
	originalGoogleSecret := os.Getenv("GOOGLE_SECRET")
	originalGitHubKey := os.Getenv("GITHUB_KEY")
	originalGitHubSecret := os.Getenv("GITHUB_SECRET")
	originalFacebookKey := os.Getenv("FACEBOOK_KEY")
	originalFacebookSecret := os.Getenv("FACEBOOK_SECRET")
	originalTwitterKey := os.Getenv("TWITTER_KEY")
	originalTwitterSecret := os.Getenv("TWITTER_SECRET")

	// Clean up after test
	defer func() {
		_ = os.Setenv("OAUTH_MOCK_SERVER_URL", originalMockURL)
		_ = os.Setenv("GOOGLE_KEY", originalGoogleKey)
		_ = os.Setenv("GOOGLE_SECRET", originalGoogleSecret)
		_ = os.Setenv("GITHUB_KEY", originalGitHubKey)
		_ = os.Setenv("GITHUB_SECRET", originalGitHubSecret)
		_ = os.Setenv("FACEBOOK_KEY", originalFacebookKey)
		_ = os.Setenv("FACEBOOK_SECRET", originalFacebookSecret)
		_ = os.Setenv("TWITTER_KEY", originalTwitterKey)
		_ = os.Setenv("TWITTER_SECRET", originalTwitterSecret)
	}()

	// Set up mock environment with all four providers
	_ = os.Setenv("OAUTH_MOCK_SERVER_URL", "http://localhost:8080")
	_ = os.Setenv("GOOGLE_KEY", "mock-google-key")
	_ = os.Setenv("GOOGLE_SECRET", "mock-google-secret")
	_ = os.Setenv("GITHUB_KEY", "mock-github-key")
	_ = os.Setenv("GITHUB_SECRET", "mock-github-secret")
	_ = os.Setenv("FACEBOOK_KEY", "mock-facebook-key")
	_ = os.Setenv("FACEBOOK_SECRET", "mock-facebook-secret")
	_ = os.Setenv("TWITTER_KEY", "mock-twitter-key")
	_ = os.Setenv("TWITTER_SECRET", "mock-twitter-secret")

	callbackTemplate := "http://localhost:3000/auth/%s/callback"

	t.Run("setup all mock providers", func(t *testing.T) {
		providers := setupMockProviders(callbackTemplate)

		if len(providers) != 4 {
			t.Fatalf("Expected 4 providers, got %d", len(providers))
		}

		// Verify we have all four providers
		providerNames := make(map[string]bool)
		for _, provider := range providers {
			providerNames[provider.Name()] = true
		}

		expectedProviders := []string{"google", "github", "facebook", "twitter"}
		for _, expected := range expectedProviders {
			if !providerNames[expected] {
				t.Errorf("Expected %s provider to be created", expected)
			}
		}
	})

	t.Run("setup only facebook and twitter providers", func(t *testing.T) {
		// Clear Google and GitHub keys
		_ = os.Setenv("GOOGLE_KEY", "")
		_ = os.Setenv("GITHUB_KEY", "")

		providers := setupMockProviders(callbackTemplate)

		if len(providers) != 2 {
			t.Fatalf("Expected 2 providers, got %d", len(providers))
		}

		// Verify we have facebook and twitter providers
		providerNames := make(map[string]bool)
		for _, provider := range providers {
			providerNames[provider.Name()] = true
		}

		if !providerNames["facebook"] {
			t.Error("Expected facebook provider to be created")
		}

		if !providerNames["twitter"] {
			t.Error("Expected twitter provider to be created")
		}
	})

	t.Run("no providers when no keys", func(t *testing.T) {
		_ = os.Setenv("GOOGLE_KEY", "")
		_ = os.Setenv("GITHUB_KEY", "")
		_ = os.Setenv("FACEBOOK_KEY", "")
		_ = os.Setenv("TWITTER_KEY", "")

		providers := setupMockProviders(callbackTemplate)

		if len(providers) != 0 {
			t.Errorf("Expected 0 providers when no keys set, got %d", len(providers))
		}
	})
}

func TestProvidersWithMockMode(t *testing.T) {
	// Save original environment
	originalMockURL := os.Getenv("OAUTH_MOCK_SERVER_URL")
	originalGoogleKey := os.Getenv("GOOGLE_KEY")
	originalGoogleSecret := os.Getenv("GOOGLE_SECRET")

	// Clean up after test
	defer func() {
		_ = os.Setenv("OAUTH_MOCK_SERVER_URL", originalMockURL)
		_ = os.Setenv("GOOGLE_KEY", originalGoogleKey)
		_ = os.Setenv("GOOGLE_SECRET", originalGoogleSecret)
		goth.ClearProviders()
	}()

	tests := []struct {
		name          string
		mockServerURL string
		googleKey     string
		expectMock    bool
	}{
		{
			name:          "production mode - no mock server",
			mockServerURL: "",
			googleKey:     "real-google-key",
			expectMock:    false,
		},
		{
			name:          "test mode - with mock server",
			mockServerURL: "http://localhost:8080",
			googleKey:     "mock-google-key",
			expectMock:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear providers before each test
			goth.ClearProviders()

			// Set up environment
			_ = os.Setenv("OAUTH_MOCK_SERVER_URL", tt.mockServerURL)
			_ = os.Setenv("GOOGLE_KEY", tt.googleKey)
			_ = os.Setenv("GOOGLE_SECRET", "mock-secret")

			// Call the providers function like the real application would
			providers("http://localhost:3000")

			// Get the providers that were registered
			registeredProviders := goth.GetProviders()

			if tt.expectMock {
				// In mock mode, we should only have the mock providers
				if len(registeredProviders) == 0 {
					t.Error("Expected mock providers to be registered, but none found")
				}

				// Verify we have a google provider
				if _, exists := registeredProviders["google"]; !exists {
					t.Error("Expected google provider to be registered in mock mode")
				}
			} else {
				// In production mode, we should have all the real providers
				// (Only if the required environment variables are set)
				// Since we only set GOOGLE_KEY, we expect the google provider to be registered
				if tt.googleKey != "" {
					if _, exists := registeredProviders["google"]; !exists {
						t.Error("Expected google provider to be registered in production mode")
					}
				}
			}
		})
	}
}
