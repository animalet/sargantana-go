# Authentication Providers Configuration

This document lists all supported authentication providers and their configuration in Sargantana-Go.

## Overview

Sargantana-Go supports **50+ authentication providers** through the [Goth library](https://github.com/markbates/goth). Each provider is configured within the `controllers` section of your YAML configuration file under an auth controller.

## Configuration

To enable and configure providers, you need to add them to the `providers` map in your auth controller configuration. The key of the map is the provider ID (e.g., `github`, `google`), and the value is an object containing the provider's credentials and other settings.

Here is an example for configuring GitHub and Google:

```yaml
controllers:
  - type: "auth"
    name: "authentication"
    config:
      # Optional: Custom callback URL (if running behind a proxy)
      callback_host: "https://myapp.example.com"
      
      # Authentication paths (optional, these are defaults)
      callback_path: "/auth/{provider}/callback"
      login_path: "/auth/{provider}"
      logout_path: "/auth/logout"
      user_info_path: "/auth/user"
      redirect_on_login: "/"
      redirect_on_logout: "/"
      
      # OAuth providers configuration
      providers:
        github:
          key: "${GITHUB_KEY}"
          secret: "${GITHUB_SECRET}"
          scopes:
            - "read:user"
            - "user:email"
        google:
          key: "${GOOGLE_KEY}"
          secret: "${GOOGLE_SECRET}"
```

### Provider Configuration Fields

Each provider is configured using the following fields:

-   `key`: The client ID or key for the provider.
-   `secret`: The client secret for the provider.
-   `scopes`: (Optional) A list of scopes to request from the provider.
-   `url`: (Optional) Required for providers like OpenID Connect and Nextcloud.
-   `domain`: (Optional) Required for Auth0.
-   `org_url`: (Optional) Required for Okta.
-   `corp_id`: (Optional) Required for WeCom.
-   `agent_id`: (Optional) Required for WeCom.

If the `key` for a provider is not set, the provider will be disabled.

## Supported Providers

The following table lists the supported providers and their unique configuration requirements. Most providers only require a `key` and a `secret`.

| Provider             | Provider ID       | Special Fields                               | Notes                                           |
|----------------------|-------------------|----------------------------------------------|-------------------------------------------------|
| **Amazon**           | `amazon`          | -                                            | -                                               |
| **Apple**            | `apple`           | -                                            | Includes name and email scopes                  |
| **Auth0**            | `auth0`           | `domain`                                     | Domain is required for Auth0                    |
| **Azure AD**         | `azuread`         | -                                            | Microsoft Azure Active Directory                |
| **Battle.net**       | `battlenet`       | -                                            | Blizzard Entertainment                          |
| **Bitbucket**        | `bitbucket`       | -                                            | Atlassian Bitbucket                             |
| **Box**              | `box`             | -                                            | Box cloud storage                               |
| **Dailymotion**      | `dailymotion`     | -                                            | Includes email scope                            |
| **Deezer**           | `deezer`          | -                                            | Includes email scope                            |
| **DigitalOcean**     | `digitalocean`    | -                                            | Includes read scope                             |
| **Discord**          | `discord`         | -                                            | Includes identify and email scopes              |
| **Dropbox**          | `dropbox`         | -                                            | -                                               |
| **EVE Online**       | `eveonline`       | -                                            | CCP Games                                       |
| **Facebook**         | `facebook`        | -                                            | Includes email and public_profile scopes        |
| **Fitbit**           | `fitbit`          | -                                            | -                                               |
| **Gitea**            | `gitea`           | -                                            | Self-hosted Git service                         |
| **GitHub**           | `github`          | -                                            | Includes read:user and user:email scopes        |
| **GitLab**           | `gitlab`          | -                                            | -                                               |
| **Google**           | `google`          | -                                            | -                                               |
| **Heroku**           | `heroku`          | -                                            | -                                               |
| **Instagram**        | `instagram`       | -                                            | -                                               |
| **Intercom**         | `intercom`        | -                                            | -                                               |
| **Kakao**            | `kakao`           | -                                            | Korean social platform                          |
| **Last.fm**          | `lastfm`          | -                                            | Music platform                                  |
| **LINE**             | `line`            | -                                            | Includes profile, openid, and email scopes      |
| **LinkedIn**         | `linkedin`        | -                                            | -                                               |
| **Mastodon**         | `mastodon`        | -                                            | Includes read:accounts scope                    |
| **Meetup**           | `meetup`          | -                                            | -                                               |
| **Microsoft Online** | `microsoftonline` | -                                            | Microsoft 365                                   |
| **Naver**            | `naver`           | -                                            | Korean search engine                            |
| **Nextcloud**        | `nextcloud`       | `url`                                        | Self-hosted cloud platform                      |
| **Okta**             | `okta`            | `org_url`                                    | Enterprise identity platform                    |
| **OneDrive**         | `onedrive`        | -                                            | Microsoft OneDrive                              |
| **OpenID Connect**   | `openid-connect`  | `url`                                        | Generic OpenID Connect provider                 |
| **Patreon**          | `patreon`         | -                                            | Creator funding platform                        |
| **PayPal**           | `paypal`          | -                                            | Set `PAYPAL_ENV=sandbox` for testing            |
| **Salesforce**       | `salesforce`      | -                                            | CRM platform                                    |
| **Seatalk**          | `seatalk`         | -                                            | -                                               |
| **Shopify**          | `shopify`         | -                                            | Includes read customers and orders scopes       |
| **Slack**            | `slack`           | -                                            | -                                               |
| **SoundCloud**       | `soundcloud`      | -                                            | -                                               |
| **Spotify**          | `spotify`         | -                                            | -                                               |
| **Steam**            | `steam`           | -                                            | Only requires API key, no secret                |
| **Strava**           | `strava`          | -                                            | Fitness tracking platform                       |
| **Stripe**           | `stripe`          | -                                            | Payment processing                              |
| **TikTok**           | `tiktok`          | -                                            | -                                               |
| **Twitch**           | `twitch`          | -                                            | -                                               |
| **Twitter v2**       | `twitterv2`       | -                                            | Uses Twitter API v2 (Essential tier compatible) |
| **Typetalk**         | `typetalk`        | -                                            | Includes "my" scope                             |
| **Uber**             | `uber`            | -                                            | -                                               |
| **VK**               | `vk`              | -                                            | Russian social network                          |
| **WeCom**            | `wecom`           | `corp_id`, `agent_id`                        | WeChat Work (enterprise)                        |
| **WePay**            | `wepay`           | -                                            | Includes view_user scope                        |
| **Xero**             | `xero`            | -                                            | Accounting software                             |
| **Yahoo**            | `yahoo`           | -                                            | ⚠️ Uses hardcoded HTTPS callback                |
| **Yammer**           | `yammer`          | -                                            | Microsoft Yammer                                |
| **Yandex**           | `yandex`          | -                                            | Russian search engine                           |
| **Zoom**             | `zoom`            | -                                            | Includes read:user scope                        |


## Custom Authentication Strategies

While Goth is the primary implementation for OAuth2, Sargantana-Go uses an interface-based approach for authentication. This allows you to implement any authentication strategy (JWT, API Keys, LDAP, etc.) by implementing the `server.Authenticator` interface.

### The Authenticator Interface

The `Authenticator` interface is defined in `pkg/server/authenticator.go`:

```go
type Authenticator interface {
    // Middleware returns a Gin middleware function that performs authentication.
    // This middleware will be called for routes that require authentication.
    Middleware() gin.HandlerFunc
}
```

### Using a Custom Authenticator

To use a custom authenticator, you need to:

1.  Implement the `Authenticator` interface.
2.  Set your authenticator using `server.SetAuthenticator()` before starting the server.

```go
// 1. Implement the interface
type MyCustomAuth struct {}

func (a *MyCustomAuth) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Your custom authentication logic here
        token := c.GetHeader("Authorization")
        if validate(token) {
            c.Next()
        } else {
            c.AbortWithStatus(http.StatusUnauthorized)
        }
    }
}

// 2. Set it in your main.go
func main() {
    // ... load config ...
    
    srv := server.NewServer(cfg)
    
    // Replace the default Goth authenticator with your custom one
    srv.SetAuthenticator(&MyCustomAuth{})
    
    // ... start server ...
}
```

### Default Behavior

By default, if no authenticator is set, the server uses `UnauthorizedAuthenticator`, which rejects **all** requests to protected routes with a 401 Unauthorized status. This is a security-by-default measure to ensure you explicitly configure authentication.

To use the standard Goth-based OAuth2 authentication (as configured in YAML), you must explicitly set it:

```go
// Use the built-in Goth authenticator
srv.SetAuthenticator(controller.NewGothAuthenticator())
```

