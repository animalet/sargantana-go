package controller

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/animalet/sargantana-go/config"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/amazon"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/auth0"
	"github.com/markbates/goth/providers/azuread"
	"github.com/markbates/goth/providers/battlenet"
	"github.com/markbates/goth/providers/bitbucket"
	"github.com/markbates/goth/providers/box"
	"github.com/markbates/goth/providers/dailymotion"
	"github.com/markbates/goth/providers/deezer"
	"github.com/markbates/goth/providers/digitalocean"
	"github.com/markbates/goth/providers/discord"
	"github.com/markbates/goth/providers/dropbox"
	"github.com/markbates/goth/providers/eveonline"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/fitbit"
	"github.com/markbates/goth/providers/gitea"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/heroku"
	"github.com/markbates/goth/providers/instagram"
	"github.com/markbates/goth/providers/intercom"
	"github.com/markbates/goth/providers/kakao"
	"github.com/markbates/goth/providers/lastfm"
	"github.com/markbates/goth/providers/line"
	"github.com/markbates/goth/providers/linkedin"
	"github.com/markbates/goth/providers/mastodon"
	"github.com/markbates/goth/providers/meetup"
	"github.com/markbates/goth/providers/microsoftonline"
	"github.com/markbates/goth/providers/naver"
	"github.com/markbates/goth/providers/nextcloud"
	"github.com/markbates/goth/providers/okta"
	"github.com/markbates/goth/providers/onedrive"
	"github.com/markbates/goth/providers/patreon"
	"github.com/markbates/goth/providers/paypal"
	"github.com/markbates/goth/providers/salesforce"
	"github.com/markbates/goth/providers/seatalk"
	"github.com/markbates/goth/providers/shopify"
	"github.com/markbates/goth/providers/slack"
	"github.com/markbates/goth/providers/soundcloud"
	"github.com/markbates/goth/providers/spotify"
	"github.com/markbates/goth/providers/steam"
	"github.com/markbates/goth/providers/strava"
	"github.com/markbates/goth/providers/stripe"
	"github.com/markbates/goth/providers/tiktok"
	"github.com/markbates/goth/providers/twitch"
	"github.com/markbates/goth/providers/twitterv2"
	"github.com/markbates/goth/providers/typetalk"
	"github.com/markbates/goth/providers/uber"
	"github.com/markbates/goth/providers/vk"
	"github.com/markbates/goth/providers/wecom"
	"github.com/markbates/goth/providers/wepay"
	"github.com/markbates/goth/providers/xero"
	"github.com/markbates/goth/providers/yahoo"
	"github.com/markbates/goth/providers/yammer"
	"github.com/markbates/goth/providers/yandex"
	"github.com/markbates/goth/providers/zoom"
)

// ProviderFactory is an interface for creating OAuth providers
type ProviderFactory interface {
	CreateProviders(callbackURLTemplate string) []goth.Provider
}

// ProductionProviderFactory creates real OAuth providers for production use
type ProductionProviderFactory struct{}

// CreateProviders creates all production OAuth providers
func (f *ProductionProviderFactory) CreateProviders(callbackURLTemplate string) []goth.Provider {
	var providers []goth.Provider

	// Use twitterv2 instead of twitter if you only have access to the Essential API Level
	if v := os.Getenv("TWITTER_KEY"); v != "" {
		providers = append(providers, twitterv2.New(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitterv2")))
	}
	// If you'd like to use authenticate instead of authorize in TwitterV2 provider, use this instead.
	//if v := os.Getenv("TWITTER_KEY"); v != "" {
	//    providers = append(providers, twitterv2.NewAuthenticate(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitterv2")))
	//}
	if v := os.Getenv("TIKTOK_KEY"); v != "" {
		providers = append(providers, tiktok.New(os.Getenv("TIKTOK_KEY"), os.Getenv("TIKTOK_SECRET"), fmt.Sprintf(callbackURLTemplate, "tiktok")))
	}
	if v := os.Getenv("FACEBOOK_KEY"); v != "" {
		providers = append(providers, facebook.New(os.Getenv("FACEBOOK_KEY"), os.Getenv("FACEBOOK_SECRET"), fmt.Sprintf(callbackURLTemplate, "facebook"), "email", "public_profile"))
	}
	if v := os.Getenv("FITBIT_KEY"); v != "" {
		providers = append(providers, fitbit.New(os.Getenv("FITBIT_KEY"), os.Getenv("FITBIT_SECRET"), fmt.Sprintf(callbackURLTemplate, "fitbit")))
	}
	if v := os.Getenv("GOOGLE_KEY"); v != "" {
		providers = append(providers, google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), fmt.Sprintf(callbackURLTemplate, "google")))
	}
	if v := os.Getenv("GITHUB_KEY"); v != "" {
		providers = append(providers, github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), fmt.Sprintf(callbackURLTemplate, "github"), "read:user", "user:email"))
	}
	if v := os.Getenv("SPOTIFY_KEY"); v != "" {
		providers = append(providers, spotify.New(os.Getenv("SPOTIFY_KEY"), os.Getenv("SPOTIFY_SECRET"), fmt.Sprintf(callbackURLTemplate, "spotify")))
	}
	if v := os.Getenv("LINKEDIN_KEY"); v != "" {
		providers = append(providers, linkedin.New(os.Getenv("LINKEDIN_KEY"), os.Getenv("LINKEDIN_SECRET"), fmt.Sprintf(callbackURLTemplate, "linkedin")))
	}
	if v := os.Getenv("LINE_KEY"); v != "" {
		providers = append(providers, line.New(os.Getenv("LINE_KEY"), os.Getenv("LINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "line"), "profile", "openid", "email"))
	}
	if v := os.Getenv("LASTFM_KEY"); v != "" {
		providers = append(providers, lastfm.New(os.Getenv("LASTFM_KEY"), os.Getenv("LASTFM_SECRET"), fmt.Sprintf(callbackURLTemplate, "lastfm")))
	}
	if v := os.Getenv("TWITCH_KEY"); v != "" {
		providers = append(providers, twitch.New(os.Getenv("TWITCH_KEY"), os.Getenv("TWITCH_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitch")))
	}
	if v := os.Getenv("DROPBOX_KEY"); v != "" {
		providers = append(providers, dropbox.New(os.Getenv("DROPBOX_KEY"), os.Getenv("DROPBOX_SECRET"), fmt.Sprintf(callbackURLTemplate, "dropbox")))
	}
	if v := os.Getenv("DIGITALOCEAN_KEY"); v != "" {
		providers = append(providers, digitalocean.New(os.Getenv("DIGITALOCEAN_KEY"), os.Getenv("DIGITALOCEAN_SECRET"), fmt.Sprintf(callbackURLTemplate, "digitalocean"), "read"))
	}
	if v := os.Getenv("BITBUCKET_KEY"); v != "" {
		providers = append(providers, bitbucket.New(os.Getenv("BITBUCKET_KEY"), os.Getenv("BITBUCKET_SECRET"), fmt.Sprintf(callbackURLTemplate, "bitbucket")))
	}
	if v := os.Getenv("INSTAGRAM_KEY"); v != "" {
		providers = append(providers, instagram.New(os.Getenv("INSTAGRAM_KEY"), os.Getenv("INSTAGRAM_SECRET"), fmt.Sprintf(callbackURLTemplate, "instagram")))
	}
	if v := os.Getenv("INTERCOM_KEY"); v != "" {
		providers = append(providers, intercom.New(os.Getenv("INTERCOM_KEY"), os.Getenv("INTERCOM_SECRET"), fmt.Sprintf(callbackURLTemplate, "intercom")))
	}
	if v := os.Getenv("BOX_KEY"); v != "" {
		providers = append(providers, box.New(os.Getenv("BOX_KEY"), os.Getenv("BOX_SECRET"), fmt.Sprintf(callbackURLTemplate, "box")))
	}
	if v := os.Getenv("SALESFORCE_KEY"); v != "" {
		providers = append(providers, salesforce.New(os.Getenv("SALESFORCE_KEY"), os.Getenv("SALESFORCE_SECRET"), fmt.Sprintf(callbackURLTemplate, "salesforce")))
	}
	if v := os.Getenv("SEATALK_KEY"); v != "" {
		providers = append(providers, seatalk.New(os.Getenv("SEATALK_KEY"), os.Getenv("SEATALK_SECRET"), fmt.Sprintf(callbackURLTemplate, "seatalk")))
	}
	if v := os.Getenv("AMAZON_KEY"); v != "" {
		providers = append(providers, amazon.New(os.Getenv("AMAZON_KEY"), os.Getenv("AMAZON_SECRET"), fmt.Sprintf(callbackURLTemplate, "amazon")))
	}
	if v := os.Getenv("YAMMER_KEY"); v != "" {
		providers = append(providers, yammer.New(os.Getenv("YAMMER_KEY"), os.Getenv("YAMMER_SECRET"), fmt.Sprintf(callbackURLTemplate, "yammer")))
	}
	if v := os.Getenv("ONEDRIVE_KEY"); v != "" {
		providers = append(providers, onedrive.New(os.Getenv("ONEDRIVE_KEY"), os.Getenv("ONEDRIVE_SECRET"), fmt.Sprintf(callbackURLTemplate, "onedrive")))
	}
	if v := os.Getenv("AZUREAD_KEY"); v != "" {
		providers = append(providers, azuread.New(os.Getenv("AZUREAD_KEY"), os.Getenv("AZUREAD_SECRET"), fmt.Sprintf(callbackURLTemplate, "azuread"), nil))
	}
	if v := os.Getenv("MICROSOFTONLINE_KEY"); v != "" {
		providers = append(providers, microsoftonline.New(os.Getenv("MICROSOFTONLINE_KEY"), os.Getenv("MICROSOFTONLINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "microsoftonline")))
	}
	if v := os.Getenv("BATTLENET_KEY"); v != "" {
		providers = append(providers, battlenet.New(os.Getenv("BATTLENET_KEY"), os.Getenv("BATTLENET_SECRET"), fmt.Sprintf(callbackURLTemplate, "battlenet")))
	}
	if v := os.Getenv("EVEONLINE_KEY"); v != "" {
		providers = append(providers, eveonline.New(os.Getenv("EVEONLINE_KEY"), os.Getenv("EVEONLINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "eveonline")))
	}
	if v := os.Getenv("KAKAO_KEY"); v != "" {
		providers = append(providers, kakao.New(os.Getenv("KAKAO_KEY"), os.Getenv("KAKAO_SECRET"), fmt.Sprintf(callbackURLTemplate, "kakao")))
	}

	// Pointed https://localhost.com to http://localhost:3000/auth/yahoo/callback
	// Yahoo only accepts urls that starts with https
	if v := os.Getenv("YAHOO_KEY"); v != "" {
		providers = append(providers, yahoo.New(os.Getenv("YAHOO_KEY"), os.Getenv("YAHOO_SECRET"), "https://localhost.com"))
	}
	if v := os.Getenv("TYPETALK_KEY"); v != "" {
		providers = append(providers, typetalk.New(os.Getenv("TYPETALK_KEY"), os.Getenv("TYPETALK_SECRET"), fmt.Sprintf(callbackURLTemplate, "typetalk"), "my"))
	}
	if v := os.Getenv("SLACK_KEY"); v != "" {
		providers = append(providers, slack.New(os.Getenv("SLACK_KEY"), os.Getenv("SLACK_SECRET"), fmt.Sprintf(callbackURLTemplate, "slack")))
	}
	if v := os.Getenv("STRIPE_KEY"); v != "" {
		providers = append(providers, stripe.New(os.Getenv("STRIPE_KEY"), os.Getenv("STRIPE_SECRET"), fmt.Sprintf(callbackURLTemplate, "stripe")))
	}
	if v := os.Getenv("WEPAY_KEY"); v != "" {
		providers = append(providers, wepay.New(os.Getenv("WEPAY_KEY"), os.Getenv("WEPAY_SECRET"), fmt.Sprintf(callbackURLTemplate, "wepay"), "view_user"))
	}
	// By default paypal production auth urls will be used, please set PAYPAL_ENV=sandbox as environment variable for testing
	// in sandbox environment
	if v := os.Getenv("PAYPAL_KEY"); v != "" {
		providers = append(providers, paypal.New(os.Getenv("PAYPAL_KEY"), os.Getenv("PAYPAL_SECRET"), fmt.Sprintf(callbackURLTemplate, "paypal")))
	}
	if v := os.Getenv("STEAM_KEY"); v != "" {
		providers = append(providers, steam.New(os.Getenv("STEAM_KEY"), fmt.Sprintf(callbackURLTemplate, "steam")))
	}
	if v := os.Getenv("HEROKU_KEY"); v != "" {
		providers = append(providers, heroku.New(os.Getenv("HEROKU_KEY"), os.Getenv("HEROKU_SECRET"), fmt.Sprintf(callbackURLTemplate, "heroku")))
	}
	if v := os.Getenv("UBER_KEY"); v != "" {
		providers = append(providers, uber.New(os.Getenv("UBER_KEY"), os.Getenv("UBER_SECRET"), fmt.Sprintf(callbackURLTemplate, "uber")))
	}
	if v := os.Getenv("SOUNDCLOUD_KEY"); v != "" {
		providers = append(providers, soundcloud.New(os.Getenv("SOUNDCLOUD_KEY"), os.Getenv("SOUNDCLOUD_SECRET"), fmt.Sprintf(callbackURLTemplate, "soundcloud")))
	}
	if v := os.Getenv("GITLAB_KEY"); v != "" {
		providers = append(providers, gitlab.New(os.Getenv("GITLAB_KEY"), os.Getenv("GITLAB_SECRET"), fmt.Sprintf(callbackURLTemplate, "gitlab")))
	}
	if v := os.Getenv("DAILYMOTION_KEY"); v != "" {
		providers = append(providers, dailymotion.New(os.Getenv("DAILYMOTION_KEY"), os.Getenv("DAILYMOTION_SECRET"), fmt.Sprintf(callbackURLTemplate, "dailymotion"), "email"))
	}
	if v := os.Getenv("DEEZER_KEY"); v != "" {
		providers = append(providers, deezer.New(os.Getenv("DEEZER_KEY"), os.Getenv("DEEZER_SECRET"), fmt.Sprintf(callbackURLTemplate, "deezer"), "email"))
	}
	if v := os.Getenv("DISCORD_KEY"); v != "" {
		providers = append(providers, discord.New(os.Getenv("DISCORD_KEY"), os.Getenv("DISCORD_SECRET"), fmt.Sprintf(callbackURLTemplate, "discord"), discord.ScopeIdentify, discord.ScopeEmail))
	}
	if v := os.Getenv("MEETUP_KEY"); v != "" {
		providers = append(providers, meetup.New(os.Getenv("MEETUP_KEY"), os.Getenv("MEETUP_SECRET"), fmt.Sprintf(callbackURLTemplate, "meetup")))
	}

	// Auth0 allocates domain per customer, a domain must be provided for auth0 to work
	if v := os.Getenv("AUTH0_KEY"); v != "" {
		providers = append(providers, auth0.New(os.Getenv("AUTH0_KEY"), os.Getenv("AUTH0_SECRET"), fmt.Sprintf(callbackURLTemplate, "auth0"), os.Getenv("AUTH0_DOMAIN")))
	}
	if v := os.Getenv("XERO_KEY"); v != "" {
		providers = append(providers, xero.New(os.Getenv("XERO_KEY"), os.Getenv("XERO_SECRET"), fmt.Sprintf(callbackURLTemplate, "xero")))
	}
	if v := os.Getenv("VK_KEY"); v != "" {
		providers = append(providers, vk.New(os.Getenv("VK_KEY"), os.Getenv("VK_SECRET"), fmt.Sprintf(callbackURLTemplate, "vk")))
	}
	if v := os.Getenv("NAVER_KEY"); v != "" {
		providers = append(providers, naver.New(os.Getenv("NAVER_KEY"), os.Getenv("NAVER_SECRET"), fmt.Sprintf(callbackURLTemplate, "naver")))
	}
	if v := os.Getenv("YANDEX_KEY"); v != "" {
		providers = append(providers, yandex.New(os.Getenv("YANDEX_KEY"), os.Getenv("YANDEX_SECRET"), fmt.Sprintf(callbackURLTemplate, "yandex")))
	}
	if v := os.Getenv("NEXTCLOUD_KEY"); v != "" {
		providers = append(providers, nextcloud.NewCustomisedDNS(os.Getenv("NEXTCLOUD_KEY"), os.Getenv("NEXTCLOUD_SECRET"), fmt.Sprintf(callbackURLTemplate, "nextcloud"), os.Getenv("NEXTCLOUD_URL")))
	}

	if v := os.Getenv("GITEA_KEY"); v != "" {
		providers = append(providers, gitea.New(os.Getenv("GITEA_KEY"), os.Getenv("GITEA_SECRET"), fmt.Sprintf(callbackURLTemplate, "gitea")))
	}
	if v := os.Getenv("SHOPIFY_KEY"); v != "" {
		providers = append(providers, shopify.New(os.Getenv("SHOPIFY_KEY"), os.Getenv("SHOPIFY_SECRET"), fmt.Sprintf(callbackURLTemplate, "shopify"), shopify.ScopeReadCustomers, shopify.ScopeReadOrders))
	}
	if v := os.Getenv("APPLE_KEY"); v != "" {
		providers = append(providers, apple.New(os.Getenv("APPLE_KEY"), os.Getenv("APPLE_SECRET"), fmt.Sprintf(callbackURLTemplate, "apple"), nil, apple.ScopeName, apple.ScopeEmail))
	}
	if v := os.Getenv("STRAVA_KEY"); v != "" {
		providers = append(providers, strava.New(os.Getenv("STRAVA_KEY"), os.Getenv("STRAVA_SECRET"), fmt.Sprintf(callbackURLTemplate, "strava")))
	}
	if v := os.Getenv("OKTA_ID"); v != "" {
		providers = append(providers, okta.New(os.Getenv("OKTA_ID"), os.Getenv("OKTA_SECRET"), os.Getenv("OKTA_ORG_URL"), fmt.Sprintf(callbackURLTemplate, "okta"), "openid", "profile", "email"))
	}
	if v := os.Getenv("MASTODON_KEY"); v != "" {
		providers = append(providers, mastodon.New(os.Getenv("MASTODON_KEY"), os.Getenv("MASTODON_SECRET"), fmt.Sprintf(callbackURLTemplate, "mastodon"), "read:accounts"))
	}
	if v := os.Getenv("WECOM_CORP_ID"); v != "" {
		providers = append(providers, wecom.New(os.Getenv("WECOM_CORP_ID"), os.Getenv("WECOM_SECRET"), os.Getenv("WECOM_AGENT_ID"), fmt.Sprintf(callbackURLTemplate, "wecom")))
	}
	if v := os.Getenv("ZOOM_KEY"); v != "" {
		providers = append(providers, zoom.New(os.Getenv("ZOOM_KEY"), os.Getenv("ZOOM_SECRET"), fmt.Sprintf(callbackURLTemplate, "zoom"), "read:user"))
	}
	if v := os.Getenv("PATREON_KEY"); v != "" {
		providers = append(providers, patreon.New(os.Getenv("PATREON_KEY"), os.Getenv("PATREON_SECRET"), fmt.Sprintf(callbackURLTemplate, "patreon")))
	}

	return providers
}

// DefaultProviderFactory returns the default provider factory for production use
func DefaultProviderFactory() ProviderFactory {
	return &ProductionProviderFactory{}
}

// Auth is a controller that provides OAuth2 authentication functionality.
// It supports 50+ OAuth2 providers through the Goth library and handles
// the complete authentication flow including user session management.
type Auth struct {
	IController
	callbackAddress string
}

// UserObject represents an authenticated user stored in the session.
// It contains both a unique identifier and the complete user information
// received from the OAuth2 provider.
type UserObject struct {
	Id   string    `json:"id"`   // Unique identifier for the user session
	User goth.User `json:"user"` // Complete user information from OAuth2 provider
}

func providers(callbackEndpoint string) {
	providersWithFactory(callbackEndpoint, DefaultProviderFactory())
}

func providersWithFactory(callbackEndpoint string, factory ProviderFactory) {
	gob.Register(UserObject{})
	callbackURLTemplate := callbackEndpoint + "/auth/%s/callback"

	providers := factory.CreateProviders(callbackURLTemplate)
	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}
}

// NewAuth creates a new Auth controller with the specified callback endpoint.
// The callback endpoint is used to construct OAuth2 callback URLs for all providers.
// If an empty string is provided, the callback endpoint will be automatically
// determined from the server configuration during binding.
//
// Parameters:
//   - callbackEndpoint: The base URL where OAuth2 providers should redirect after authentication
//     (e.g., "https://myapp.com" will result in callbacks to "https://myapp.com/auth/{provider}/callback")
//
// Returns a pointer to the configured Auth controller.
func NewAuth(callbackEndpoint string) *Auth {
	return &Auth{callbackAddress: callbackEndpoint}
}

// NewAuthFromFlags creates an Auth controller factory function that reads
// configuration from command-line flags. This function is designed to be used
// with the server's flag-based initialization system.
//
// The following flags are registered:
//   - callback: Callback endpoint for authentication, used when behind a reverse proxy or load balancer
//     If not set, it will default to http://<host>:<port>
//
// Parameters:
//   - flagSet: The flag set to register the auth controller flags with
//
// Returns a factory function that creates an Auth controller when called.
func NewAuthFromFlags(flagSet *flag.FlagSet) func() IController {
	callback := flagSet.String("callback", "", "Callback endpoint for authentication, in case you are behind a reverse proxy or load balancer. If not set, it will default to http://<host>:<port>")
	return func() IController { return NewAuth(*callback) }
}

// LoginFunc is a middleware function that protects routes requiring authentication.
// It verifies that a user is logged in and their session has not expired.
// If the user is not authenticated or their session has expired, the request is aborted
// with an appropriate HTTP status code.
//
// Usage:
//
//	engine.GET("/protected", controller.LoginFunc, myProtectedHandler)
//
// Responses:
//   - 403 Forbidden: User is not logged in
//   - 401 Unauthorized: User session has expired
//   - 500 Internal Server Error: Failed to clear expired session
//   - Continues to next handler: User is authenticated and session is valid
func LoginFunc(c *gin.Context) {
	userSession := sessions.Default(c)
	userObject := userSession.Get("user")
	if userObject == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	u, ok := userObject.(UserObject)

	if !ok || time.Now().After(u.User.ExpiresAt) {
		userSession.Clear()
		saveSession(c, userSession)
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	c.Next()
}

func (a *Auth) Bind(server *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc) {
	if a.callbackAddress == "" {
		address := config.Address()
		// Add http:// if not present
		if !strings.Contains(address, "://") {
			address = "http://" + address
		}
		u, err := url.Parse(address)
		if err != nil {
			log.Panicf("Failed to parse auth callback address %q: %v", address, err)
		}
		if u.Hostname() == "0.0.0.0" {
			log.Println("Auth callback endpoint is set to 0.0.0.0, changing it to localhost")
			a.callbackAddress = u.Scheme + "://localhost" + ":" + u.Port()
		}
		a.callbackAddress = u.Scheme + "://" + u.Hostname() + ":" + u.Port()
	}

	log.Printf("Callback endpoint: %q\n", a.callbackAddress)
	providers(a.callbackAddress)

	server.Group("/auth").
		Use(func(c *gin.Context) {
			// Hack to make gothic work with gin
			q := c.Request.URL.Query()
			param := c.Param("provider")
			if param == "" {
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}
			q.Add("provider", param)
			c.Request.URL.RawQuery = q.Encode()
			c.Next()
		}).
		GET("/:provider", a.login).
		GET("/:provider/callback", a.callback).
		GET("/:provider/logout", a.logout)

	server.GET("/auth/user", loginMiddleware, func(c *gin.Context) {
		c.JSON(http.StatusOK, sessions.Default(c).Get("user").(UserObject).Id)
	})
}

func (a *Auth) Close() error {
	return nil
}

func (a *Auth) success(c *gin.Context, user goth.User) {
	session := sessions.Default(c)
	session.Set("user", a.userFactory(user))
	saveSession(c, session)
	c.Redirect(http.StatusFound, "/")
}

func (a *Auth) login(c *gin.Context) {
	if user, err := gothic.CompleteUserAuth(c.Writer, c.Request); err != nil {
		gothic.BeginAuthHandler(c.Writer, c.Request)
	} else {
		a.success(c, user)
	}
}

func (a *Auth) callback(c *gin.Context) {
	if user, err := gothic.CompleteUserAuth(c.Writer, c.Request); err != nil {
		_ = c.AbortWithError(http.StatusForbidden, err)
	} else {
		a.success(c, user)
	}
}

func (a *Auth) logout(c *gin.Context) {
	err := gothic.Logout(c.Writer, c.Request)
	if err != nil {
		log.Printf("Failed to log out: %v", err)
	}
	session := sessions.Default(c)
	session.Clear()
	saveSession(c, session)
	c.Redirect(http.StatusFound, "/")
}

func saveSession(c *gin.Context, session sessions.Session) {
	if err := session.Save(); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (a *Auth) userFactory(user goth.User) *UserObject {
	var id string
	if user.Email == "" {
		id = user.UserID + "@" + user.Provider
	} else {
		id = user.Email
	}
	return &UserObject{
		Id:   id,
		User: user,
	}
}
