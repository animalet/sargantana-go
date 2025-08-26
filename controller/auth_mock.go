package controller

import (
	"fmt"
	"os"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/twitter"
)

// setupMockProviders configures providers to use mock OAuth endpoints when testing
func setupMockProviders(callbackURLTemplate string) []goth.Provider {
	mockServerURL := os.Getenv("OAUTH_MOCK_SERVER_URL")

	if mockServerURL == "" {
		return nil
	}

	var providers []goth.Provider

	// Mock Google Provider using custom endpoints
	if os.Getenv("GOOGLE_KEY") != "" {
		googleProvider := google.New(
			os.Getenv("GOOGLE_KEY"),
			os.Getenv("GOOGLE_SECRET"),
			fmt.Sprintf(callbackURLTemplate, "google"),
			mockServerURL+"/default/authorize", // Custom auth URL
			mockServerURL+"/default/token",     // Custom token URL
			mockServerURL+"/default/userinfo",  // Custom user info URL
		)
		providers = append(providers, googleProvider)
	}

	// Mock GitHub Provider using standard constructor
	if os.Getenv("GITHUB_KEY") != "" {
		githubProvider := github.New(
			os.Getenv("GITHUB_KEY"),
			os.Getenv("GITHUB_SECRET"),
			fmt.Sprintf(callbackURLTemplate, "github"),
			"read:user", "user:email",
		)
		providers = append(providers, githubProvider)
	}

	// Mock Facebook Provider
	if os.Getenv("FACEBOOK_KEY") != "" {
		facebookProvider := facebook.New(
			os.Getenv("FACEBOOK_KEY"),
			os.Getenv("FACEBOOK_SECRET"),
			fmt.Sprintf(callbackURLTemplate, "facebook"),
			"email", "public_profile",
		)
		providers = append(providers, facebookProvider)
	}

	// Mock Twitter Provider
	if os.Getenv("TWITTER_KEY") != "" {
		twitterProvider := twitter.New(
			os.Getenv("TWITTER_KEY"),
			os.Getenv("TWITTER_SECRET"),
			fmt.Sprintf(callbackURLTemplate, "twitter"),
		)
		providers = append(providers, twitterProvider)
	}

	return providers
}

// isMockMode checks if we're running in mock mode
func isMockMode() bool {
	return os.Getenv("OAUTH_MOCK_SERVER_URL") != ""
}
