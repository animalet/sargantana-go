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
	"github.com/markbates/goth/providers/twitter"
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

type Auth struct {
	IController
	callbackAddress string
}

type UserObject struct {
	Id   string    `json:"id"`
	User goth.User `json:"user"`
}

func providers(callbackEndpoint string) {
	gob.Register(UserObject{})
	callbackURLTemplate := callbackEndpoint + "/auth/%s/callback"

	goth.UseProviders(
		// Use twitterv2 instead of twitter if you only have access to the Essential API Level
		twitterv2.New(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitterv2")),
		// If you'd like to use authenticate instead of authorize in TwitterV2 provider, use this instead.
		// twitterv2.NewAuthenticate(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitterv2")),

		twitter.New(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitter")),
		// If you'd like to use authenticate instead of authorize in Twitter provider, use this instead.
		// twitter.NewAuthenticate(os.Getenv("TWITTER_KEY"), os.Getenv("TWITTER_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitter")),

		tiktok.New(os.Getenv("TIKTOK_KEY"), os.Getenv("TIKTOK_SECRET"), fmt.Sprintf(callbackURLTemplate, "tiktok")),
		facebook.New(os.Getenv("FACEBOOK_KEY"), os.Getenv("FACEBOOK_SECRET"), fmt.Sprintf(callbackURLTemplate, "facebook"), "email", "public_profile"),
		fitbit.New(os.Getenv("FITBIT_KEY"), os.Getenv("FITBIT_SECRET"), fmt.Sprintf(callbackURLTemplate, "fitbit")),
		google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), fmt.Sprintf(callbackURLTemplate, "google")),
		//gplus.New(os.Getenv("GPLUS_KEY"), os.Getenv("GPLUS_SECRET"), fmt.Sprintf(callbackURLTemplate, "gplus")),
		github.New(os.Getenv("GITHUB_KEY"), os.Getenv("GITHUB_SECRET"), fmt.Sprintf(callbackURLTemplate, "github"), "read:user", "user:email"),
		spotify.New(os.Getenv("SPOTIFY_KEY"), os.Getenv("SPOTIFY_SECRET"), fmt.Sprintf(callbackURLTemplate, "spotify")),
		linkedin.New(os.Getenv("LINKEDIN_KEY"), os.Getenv("LINKEDIN_SECRET"), fmt.Sprintf(callbackURLTemplate, "linkedin")),
		line.New(os.Getenv("LINE_KEY"), os.Getenv("LINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "line"), "profile", "openid", "email"),
		lastfm.New(os.Getenv("LASTFM_KEY"), os.Getenv("LASTFM_SECRET"), fmt.Sprintf(callbackURLTemplate, "lastfm")),
		twitch.New(os.Getenv("TWITCH_KEY"), os.Getenv("TWITCH_SECRET"), fmt.Sprintf(callbackURLTemplate, "twitch")),
		dropbox.New(os.Getenv("DROPBOX_KEY"), os.Getenv("DROPBOX_SECRET"), fmt.Sprintf(callbackURLTemplate, "dropbox")),
		digitalocean.New(os.Getenv("DIGITALOCEAN_KEY"), os.Getenv("DIGITALOCEAN_SECRET"), fmt.Sprintf(callbackURLTemplate, "digitalocean"), "read"),
		bitbucket.New(os.Getenv("BITBUCKET_KEY"), os.Getenv("BITBUCKET_SECRET"), fmt.Sprintf(callbackURLTemplate, "bitbucket")),
		instagram.New(os.Getenv("INSTAGRAM_KEY"), os.Getenv("INSTAGRAM_SECRET"), fmt.Sprintf(callbackURLTemplate, "instagram")),
		intercom.New(os.Getenv("INTERCOM_KEY"), os.Getenv("INTERCOM_SECRET"), fmt.Sprintf(callbackURLTemplate, "intercom")),
		box.New(os.Getenv("BOX_KEY"), os.Getenv("BOX_SECRET"), fmt.Sprintf(callbackURLTemplate, "box")),
		salesforce.New(os.Getenv("SALESFORCE_KEY"), os.Getenv("SALESFORCE_SECRET"), fmt.Sprintf(callbackURLTemplate, "salesforce")),
		seatalk.New(os.Getenv("SEATALK_KEY"), os.Getenv("SEATALK_SECRET"), fmt.Sprintf(callbackURLTemplate, "seatalk")),
		amazon.New(os.Getenv("AMAZON_KEY"), os.Getenv("AMAZON_SECRET"), fmt.Sprintf(callbackURLTemplate, "amazon")),
		yammer.New(os.Getenv("YAMMER_KEY"), os.Getenv("YAMMER_SECRET"), fmt.Sprintf(callbackURLTemplate, "yammer")),
		onedrive.New(os.Getenv("ONEDRIVE_KEY"), os.Getenv("ONEDRIVE_SECRET"), fmt.Sprintf(callbackURLTemplate, "onedrive")),
		azuread.New(os.Getenv("AZUREAD_KEY"), os.Getenv("AZUREAD_SECRET"), fmt.Sprintf(callbackURLTemplate, "azuread"), nil),
		microsoftonline.New(os.Getenv("MICROSOFTONLINE_KEY"), os.Getenv("MICROSOFTONLINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "microsoftonline")),
		battlenet.New(os.Getenv("BATTLENET_KEY"), os.Getenv("BATTLENET_SECRET"), fmt.Sprintf(callbackURLTemplate, "battlenet")),
		eveonline.New(os.Getenv("EVEONLINE_KEY"), os.Getenv("EVEONLINE_SECRET"), fmt.Sprintf(callbackURLTemplate, "eveonline")),
		kakao.New(os.Getenv("KAKAO_KEY"), os.Getenv("KAKAO_SECRET"), fmt.Sprintf(callbackURLTemplate, "kakao")),

		// Pointed https://localhost.com to http://localhost:3000/auth/yahoo/callback
		// Yahoo only accepts urls that starts with https
		yahoo.New(os.Getenv("YAHOO_KEY"), os.Getenv("YAHOO_SECRET"), "https://localhost.com"),
		typetalk.New(os.Getenv("TYPETALK_KEY"), os.Getenv("TYPETALK_SECRET"), fmt.Sprintf(callbackURLTemplate, "typetalk"), "my"),
		slack.New(os.Getenv("SLACK_KEY"), os.Getenv("SLACK_SECRET"), fmt.Sprintf(callbackURLTemplate, "slack")),
		stripe.New(os.Getenv("STRIPE_KEY"), os.Getenv("STRIPE_SECRET"), fmt.Sprintf(callbackURLTemplate, "stripe")),
		wepay.New(os.Getenv("WEPAY_KEY"), os.Getenv("WEPAY_SECRET"), fmt.Sprintf(callbackURLTemplate, "wepay"), "view_user"),
		// By default paypal production auth urls will be used, please set PAYPAL_ENV=sandbox as environment variable for testing
		// in sandbox environment
		paypal.New(os.Getenv("PAYPAL_KEY"), os.Getenv("PAYPAL_SECRET"), fmt.Sprintf(callbackURLTemplate, "paypal")),
		steam.New(os.Getenv("STEAM_KEY"), fmt.Sprintf(callbackURLTemplate, "steam")),
		heroku.New(os.Getenv("HEROKU_KEY"), os.Getenv("HEROKU_SECRET"), fmt.Sprintf(callbackURLTemplate, "heroku")),
		uber.New(os.Getenv("UBER_KEY"), os.Getenv("UBER_SECRET"), fmt.Sprintf(callbackURLTemplate, "uber")),
		soundcloud.New(os.Getenv("SOUNDCLOUD_KEY"), os.Getenv("SOUNDCLOUD_SECRET"), fmt.Sprintf(callbackURLTemplate, "soundcloud")),
		gitlab.New(os.Getenv("GITLAB_KEY"), os.Getenv("GITLAB_SECRET"), fmt.Sprintf(callbackURLTemplate, "gitlab")),
		dailymotion.New(os.Getenv("DAILYMOTION_KEY"), os.Getenv("DAILYMOTION_SECRET"), fmt.Sprintf(callbackURLTemplate, "dailymotion"), "email"),
		deezer.New(os.Getenv("DEEZER_KEY"), os.Getenv("DEEZER_SECRET"), fmt.Sprintf(callbackURLTemplate, "deezer"), "email"),
		discord.New(os.Getenv("DISCORD_KEY"), os.Getenv("DISCORD_SECRET"), fmt.Sprintf(callbackURLTemplate, "discord"), discord.ScopeIdentify, discord.ScopeEmail),
		meetup.New(os.Getenv("MEETUP_KEY"), os.Getenv("MEETUP_SECRET"), fmt.Sprintf(callbackURLTemplate, "meetup")),

		// Auth0 allocates domain per customer, a domain must be provided for auth0 to work
		auth0.New(os.Getenv("AUTH0_KEY"), os.Getenv("AUTH0_SECRET"), fmt.Sprintf(callbackURLTemplate, "auth0"), os.Getenv("AUTH0_DOMAIN")),
		xero.New(os.Getenv("XERO_KEY"), os.Getenv("XERO_SECRET"), fmt.Sprintf(callbackURLTemplate, "xero")),
		vk.New(os.Getenv("VK_KEY"), os.Getenv("VK_SECRET"), fmt.Sprintf(callbackURLTemplate, "vk")),
		naver.New(os.Getenv("NAVER_KEY"), os.Getenv("NAVER_SECRET"), fmt.Sprintf(callbackURLTemplate, "naver")),
		yandex.New(os.Getenv("YANDEX_KEY"), os.Getenv("YANDEX_SECRET"), fmt.Sprintf(callbackURLTemplate, "yandex")),
		nextcloud.NewCustomisedDNS(os.Getenv("NEXTCLOUD_KEY"), os.Getenv("NEXTCLOUD_SECRET"), fmt.Sprintf(callbackURLTemplate, "nextcloud"), os.Getenv("NEXTCLOUD_URL")),
		gitea.New(os.Getenv("GITEA_KEY"), os.Getenv("GITEA_SECRET"), fmt.Sprintf(callbackURLTemplate, "gitea")),
		shopify.New(os.Getenv("SHOPIFY_KEY"), os.Getenv("SHOPIFY_SECRET"), fmt.Sprintf(callbackURLTemplate, "shopify"), shopify.ScopeReadCustomers, shopify.ScopeReadOrders),
		apple.New(os.Getenv("APPLE_KEY"), os.Getenv("APPLE_SECRET"), fmt.Sprintf(callbackURLTemplate, "apple"), nil, apple.ScopeName, apple.ScopeEmail),
		strava.New(os.Getenv("STRAVA_KEY"), os.Getenv("STRAVA_SECRET"), fmt.Sprintf(callbackURLTemplate, "strava")),
		okta.New(os.Getenv("OKTA_ID"), os.Getenv("OKTA_SECRET"), os.Getenv("OKTA_ORG_URL"), fmt.Sprintf(callbackURLTemplate, "okta"), "openid", "profile", "email"),
		mastodon.New(os.Getenv("MASTODON_KEY"), os.Getenv("MASTODON_SECRET"), fmt.Sprintf(callbackURLTemplate, "mastodon"), "read:accounts"),
		wecom.New(os.Getenv("WECOM_CORP_ID"), os.Getenv("WECOM_SECRET"), os.Getenv("WECOM_AGENT_ID"), fmt.Sprintf(callbackURLTemplate, "wecom")),
		zoom.New(os.Getenv("ZOOM_KEY"), os.Getenv("ZOOM_SECRET"), fmt.Sprintf(callbackURLTemplate, "zoom"), "read:user"),
		patreon.New(os.Getenv("PATREON_KEY"), os.Getenv("PATREON_SECRET"), fmt.Sprintf(callbackURLTemplate, "patreon")),
	)
}

func NewAuth(callbackEndpoint string) *Auth {
	return &Auth{callbackAddress: callbackEndpoint}
}

func NewAuthFromFlags(flagSet *flag.FlagSet) func() IController {
	callback := flagSet.String("callback", "", "Callback endpoint for authentication, in case you are behind a reverse proxy or load balancer. If not set, it will default to http://<host>:<port>")
	return func() IController { return NewAuth(*callback) }
}

func LoginFunc(c *gin.Context) {
	userSession := sessions.Default(c)
	userObject := userSession.Get("user")
	if userObject == nil {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if time.Now().After(userObject.(UserObject).User.ExpiresAt) {
		userSession.Clear()
		if err := userSession.Save(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		} else {
			c.AbortWithStatus(http.StatusUnauthorized)
		}
		return
	}

	c.Next()
}

func (a *Auth) Bind(server *gin.Engine, config config.Config, loginMiddleware gin.HandlerFunc) {
	if a.callbackAddress == "" {
		address := config.Address()
		// Add http:// if not present
		if strings.Index(address, "://") == -1 {
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
			q.Add("provider", c.Param("provider"))
			c.Request.URL.RawQuery = q.Encode()
			c.Next()
		}).
		GET("/:provider", a.login).
		GET("/:provider/callback", a.callback).
		GET("/:provider/logout", a.logout).
		GET("/user", loginMiddleware, func(c *gin.Context) {
			c.JSON(http.StatusOK, sessions.Default(c).Get("user").(UserObject).Id)
		})
}

func (a *Auth) Close() error {
	return nil
}

func (a *Auth) success(c *gin.Context, user goth.User) {
	session := sessions.Default(c)
	session.Set("user", a.userFactory(user))
	a.saveSession(c, session)
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
		c.AbortWithError(http.StatusForbidden, err)
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
	a.saveSession(c, session)
}

func (a *Auth) saveSession(c *gin.Context, session sessions.Session) {
	if err := session.Save(); err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	} else {
		c.Redirect(http.StatusFound, "/")
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
