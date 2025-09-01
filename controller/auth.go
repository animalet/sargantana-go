package controller

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
	"github.com/markbates/goth/providers/openidConnect"
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
	"github.com/pkg/errors"
)

func init() {
	RegisterController("auth", NewAuthController)
}

// ProviderConfig represents the configuration for an OAuth provider
type ProviderConfig struct {
	Key     string   `yaml:"key"`
	Secret  string   `yaml:"secret"`
	Scopes  []string `yaml:"scopes,omitempty"`
	URL     string   `yaml:"url,omitempty"`      // For providers like OpenID Connect, Nextcloud
	Domain  string   `yaml:"domain,omitempty"`   // For Auth0
	OrgURL  string   `yaml:"org_url,omitempty"`  // For Okta
	CorpID  string   `yaml:"corp_id,omitempty"`  // For WeCom
	AgentID string   `yaml:"agent_id,omitempty"` // For WeCom
}

type AuthControllerConfig struct {
	CallbackURL      string                    `yaml:"callback_url"`
	CallbackPath     string                    `yaml:"callback_path"`
	LoginPath        string                    `yaml:"login_path"`
	LogoutPath       string                    `yaml:"logout_path"`
	UserInfoPath     string                    `yaml:"user_info_path"`
	RedirectOnLogin  string                    `yaml:"redirect_on_login"`
	RedirectOnLogout string                    `yaml:"redirect_on_logout"`
	Providers        map[string]ProviderConfig `yaml:"providers"`
}

func NewAuthController(configData config.ControllerConfig, serverConfig config.ServerConfig) (IController, error) {
	c, err := config.UnmarshalTo[AuthControllerConfig](configData)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal auth controller config")
	}

	// Use callback_url from config if provided, otherwise build from server address
	var callbackEndpoint string
	if c.CallbackURL != "" {
		callbackEndpoint = c.CallbackURL
	} else {
		address := serverConfig.Address
		// Add http:// if not present
		if !strings.Contains(address, "://") {
			address = "http://" + address
		}
		u, err := url.Parse(address)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse address")
		}
		if u.Hostname() == "0.0.0.0" {
			log.Println("Auth callback endpoint is set to 0.0.0.0, changing it to localhost")
			callbackEndpoint = u.Scheme + "://localhost" + ":" + u.Port()
		} else {
			callbackEndpoint = u.Scheme + "://" + u.Hostname() + ":" + u.Port()
		}
	}

	callbackPath := c.CallbackPath
	callbackURLTemplate := callbackEndpoint + "/" + strings.TrimPrefix(callbackPath, "/")

	gob.Register(UserObject{})
	providerFactory := ProviderFactory
	if providerFactory == nil {
		providerFactory = &configProviderFactory{config: c.Providers}
	}
	providers := providerFactory.CreateProviders(callbackURLTemplate)
	if len(providers) > 0 {
		goth.UseProviders(providers...)
	}

	return &auth{
		loginPath:        providerToGin(c.LoginPath),
		logoutPath:       providerToGin(c.LogoutPath),
		userInfoPath:     providerToGin(c.UserInfoPath),
		redirectOnLogin:  providerToGin(c.RedirectOnLogin),
		redirectOnLogout: providerToGin(c.RedirectOnLogout),
		callbackPath:     providerToGin(callbackPath),
	}, nil
}

func providerToGin(str string) string {
	return strings.ReplaceAll(str, "{provider}", ":provider")
}

// ProvidersFactory is an interface for creating OAuth providers
type ProvidersFactory interface {
	CreateProviders(callbackURLTemplate string) []goth.Provider
}

// ProviderFactory is the global provider factory instance.
// Replace it at your convenience by assigning a new factory to it.
// If it is nil, the default production provider factory will be used.
var ProviderFactory ProvidersFactory

// configProviderFactory creates OAuth providers based on configuration
type configProviderFactory struct {
	config map[string]ProviderConfig
}

// CreateProviders creates OAuth providers from configuration
func (f *configProviderFactory) CreateProviders(callbackURLTemplate string) []goth.Provider {
	callbackURLTemplate = strings.ReplaceAll(callbackURLTemplate, "{provider}", "%s")

	var providers []goth.Provider

	for providerName, providerConfig := range f.config {
		switch providerName {
		case "twitter", "twitterv2":
			providers = append(providers, twitterv2.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "twitterv2")))
		case "tiktok":
			providers = append(providers, tiktok.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "tiktok")))
		case "facebook":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"email", "public_profile"}
			}
			providers = append(providers, facebook.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "facebook"), scopes...))
		case "fitbit":
			providers = append(providers, fitbit.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "fitbit")))
		case "google":
			providers = append(providers, google.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "google")))
		case "github":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"read:user", "user:email"}
			}
			providers = append(providers, github.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "github"), scopes...))
		case "spotify":
			providers = append(providers, spotify.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "spotify")))
		case "linkedin":
			providers = append(providers, linkedin.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "linkedin")))
		case "line":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"profile", "openid", "email"}
			}
			providers = append(providers, line.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "line"), scopes...))
		case "lastfm":
			providers = append(providers, lastfm.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "lastfm")))
		case "twitch":
			providers = append(providers, twitch.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "twitch")))
		case "dropbox":
			providers = append(providers, dropbox.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "dropbox")))
		case "digitalocean":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"read"}
			}
			providers = append(providers, digitalocean.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "digitalocean"), scopes...))
		case "bitbucket":
			providers = append(providers, bitbucket.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "bitbucket")))
		case "instagram":
			providers = append(providers, instagram.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "instagram")))
		case "intercom":
			providers = append(providers, intercom.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "intercom")))
		case "box":
			providers = append(providers, box.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "box")))
		case "salesforce":
			providers = append(providers, salesforce.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "salesforce")))
		case "seatalk":
			providers = append(providers, seatalk.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "seatalk")))
		case "amazon":
			providers = append(providers, amazon.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "amazon")))
		case "yammer":
			providers = append(providers, yammer.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "yammer")))
		case "onedrive":
			providers = append(providers, onedrive.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "onedrive")))
		case "azuread":
			providers = append(providers, azuread.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "azuread"), nil))
		case "microsoftonline":
			providers = append(providers, microsoftonline.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "microsoftonline")))
		case "battlenet":
			providers = append(providers, battlenet.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "battlenet")))
		case "eveonline":
			providers = append(providers, eveonline.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "eveonline")))
		case "kakao":
			providers = append(providers, kakao.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "kakao")))
		case "yahoo":
			providers = append(providers, yahoo.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "yahoo")))
		case "typetalk":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"my"}
			}
			providers = append(providers, typetalk.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "typetalk"), scopes...))
		case "slack":
			providers = append(providers, slack.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "slack")))
		case "stripe":
			providers = append(providers, stripe.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "stripe")))
		case "wepay":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"view_user"}
			}
			providers = append(providers, wepay.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "wepay"), scopes...))
		case "paypal":
			providers = append(providers, paypal.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "paypal")))
		case "steam":
			providers = append(providers, steam.New(providerConfig.Key, fmt.Sprintf(callbackURLTemplate, "steam")))
		case "heroku":
			providers = append(providers, heroku.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "heroku")))
		case "uber":
			providers = append(providers, uber.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "uber")))
		case "soundcloud":
			providers = append(providers, soundcloud.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "soundcloud")))
		case "gitlab":
			providers = append(providers, gitlab.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "gitlab")))
		case "dailymotion":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"email"}
			}
			providers = append(providers, dailymotion.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "dailymotion"), scopes...))
		case "deezer":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"email"}
			}
			providers = append(providers, deezer.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "deezer"), scopes...))
		case "discord":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{discord.ScopeIdentify, discord.ScopeEmail}
			}
			providers = append(providers, discord.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "discord"), scopes...))
		case "meetup":
			providers = append(providers, meetup.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "meetup")))
		case "auth0":
			providers = append(providers, auth0.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "auth0"), providerConfig.Domain))
		case "xero":
			providers = append(providers, xero.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "xero")))
		case "vk":
			providers = append(providers, vk.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "vk")))
		case "naver":
			providers = append(providers, naver.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "naver")))
		case "yandex":
			providers = append(providers, yandex.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "yandex")))
		case "nextcloud":
			providers = append(providers, nextcloud.NewCustomisedDNS(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "nextcloud"), providerConfig.URL))
		case "gitea":
			providers = append(providers, gitea.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "gitea")))
		case "shopify":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{shopify.ScopeReadCustomers, shopify.ScopeReadOrders}
			}
			providers = append(providers, shopify.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "shopify"), scopes...))
		case "apple":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{apple.ScopeName, apple.ScopeEmail}
			}
			providers = append(providers, apple.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "apple"), nil, scopes...))
		case "strava":
			providers = append(providers, strava.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "strava")))
		case "okta":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"openid", "profile", "email"}
			}
			providers = append(providers, okta.New(providerConfig.Key, providerConfig.Secret, providerConfig.OrgURL, fmt.Sprintf(callbackURLTemplate, "okta"), scopes...))
		case "mastodon":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"read:accounts"}
			}
			providers = append(providers, mastodon.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "mastodon"), scopes...))
		case "wecom":
			providers = append(providers, wecom.New(providerConfig.CorpID, providerConfig.Secret, providerConfig.AgentID, fmt.Sprintf(callbackURLTemplate, "wecom")))
		case "zoom":
			scopes := providerConfig.Scopes
			if len(scopes) == 0 {
				scopes = []string{"read:user"}
			}
			providers = append(providers, zoom.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "zoom"), scopes...))
		case "patreon":
			providers = append(providers, patreon.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "patreon")))
		case "openid-connect":
			openid, _ := openidConnect.New(providerConfig.Key, providerConfig.Secret, fmt.Sprintf(callbackURLTemplate, "openid-connect"), providerConfig.URL)
			if openid != nil {
				providers = append(providers, openid)
			}
		}
	}

	return providers
}

// auth is a controller that provides OAuth2 authentication functionality.
// It supports 50+ OAuth2 providers through the Goth library and handles
// the complete authentication flow including user session management.
type auth struct {
	IController
	loginPath        string
	logoutPath       string
	userInfoPath     string
	redirectOnLogin  string
	redirectOnLogout string
	callbackPath     string
}

// UserObject represents an authenticated user stored in the session.
// It contains both a unique identifier and the complete user information
// received from the OAuth2 provider.
type UserObject struct {
	Id   string    `json:"id"`   // Unique identifier for the user session
	User goth.User `json:"user"` // Complete user information from OAuth2 provider
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

func (a *auth) Bind(engine *gin.Engine, loginMiddleware gin.HandlerFunc) {
	engine.Group("/").
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
		GET(a.loginPath, a.login).
		GET(a.callbackPath, a.callback).
		GET(a.logoutPath, a.logout)
	engine.GET(a.userInfoPath, loginMiddleware, a.userInfo)
}

func (a *auth) Close() error {
	return nil
}

func (a *auth) success(c *gin.Context, user goth.User) {
	session := sessions.Default(c)
	session.Set("user", a.userFactory(user))
	saveSession(c, session)
	c.Redirect(http.StatusFound, a.redirectOnLogin)
}

func (a *auth) login(c *gin.Context) {
	if user, err := gothic.CompleteUserAuth(c.Writer, c.Request); err != nil {
		gothic.BeginAuthHandler(c.Writer, c.Request)
	} else {
		a.success(c, user)
	}
}

func (a *auth) callback(c *gin.Context) {
	if user, err := gothic.CompleteUserAuth(c.Writer, c.Request); err != nil {
		err = c.Error(err)
		c.JSON(http.StatusForbidden, err)
		c.Abort()
	} else {
		a.success(c, user)
	}
}

func (a *auth) logout(c *gin.Context) {
	err := gothic.Logout(c.Writer, c.Request)
	if err != nil {
		log.Printf("Failed to log out: %v", err)
	}
	session := sessions.Default(c)
	session.Clear()
	saveSession(c, session)
	c.Redirect(http.StatusFound, a.redirectOnLogout)
}

func (a *auth) userInfo(c *gin.Context) {
	c.JSON(http.StatusOK, sessions.Default(c).Get("user").(UserObject).Id)
}

func saveSession(c *gin.Context, session sessions.Session) {
	if err := session.Save(); err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (a *auth) userFactory(user goth.User) *UserObject {
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
