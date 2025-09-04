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
      callback_url: "https://myapp.example.com"
      
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

## Special Configuration Notes

### Yahoo

Yahoo has special requirements and uses a hardcoded callback URL of `https://localhost.com`. You need to configure your
Yahoo app to use this specific callback URL.

### Auth0

Auth0 requires a domain configuration. Make sure to set the `domain` field to your Auth0 domain.

```yaml
controllers:
  - type: "auth"
    config:
      providers:
        auth0:
          key: "${AUTH0_KEY}"
          secret: "${AUTH0_SECRET}"
          domain: "yourdomain.auth0.com"
```

### Okta

Okta requires the organization URL (`org_url`).

```yaml
controllers:
  - type: "auth"
    config:
      providers:
        okta:
          key: "${OKTA_KEY}"
          secret: "${OKTA_SECRET}"
          org_url: "https://yourorg.okta.com"
```

### Nextcloud

Nextcloud requires the `url` to specify your Nextcloud instance URL.

```yaml
controllers:
  - type: "auth"
    config:
      providers:
        nextcloud:
          key: "${NEXTCLOUD_KEY}"
          secret: "${NEXTCLOUD_SECRET}"
          url: "https://nextcloud.example.com"
```

### OpenID Connect

OpenID Connect requires the `url` to specify the OpenID Connect provider URL.

```yaml
controllers:
  - type: "auth"
    config:
      providers:
        openid-connect:
          key: "${OIDC_KEY}"
          secret: "${OIDC_SECRET}"
          url: "https://oidc.example.com"
```

### WeCom (WeChat Work)

WeCom requires three fields: `corp_id`, `secret`, and `agent_id`.

```yaml
controllers:
  - type: "auth"
    config:
      providers:
        wecom:
          key: "${WECOM_KEY}"
          secret: "${WECOM_SECRET}"
          corp_id: "${WECOM_CORP_ID}"
          agent_id: "${WECOM_AGENT_ID}"
```
