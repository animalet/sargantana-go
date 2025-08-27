# OAuth Providers Configuration

This document lists all supported OAuth providers and their required environment variables for authentication in
Sargantana-Go.

## Overview

Sargantana-Go supports **50+ OAuth providers** through the [Goth library](https://github.com/markbates/goth). Each
provider requires specific environment variables to be configured for authentication to work.

## Configuration

To enable a provider, set the required environment variables. If the primary environment variable (usually
`{PROVIDER}_KEY`) is not set, the provider will be disabled.

## Supported Providers

| Provider             | Provider ID       | Primary Key           | Secret Key               | Additional Variables           | Notes                                           |
|----------------------|-------------------|-----------------------|--------------------------|--------------------------------|-------------------------------------------------|
| **Amazon**           | `amazon`          | `AMAZON_KEY`          | `AMAZON_SECRET`          | -                              | -                                               |
| **Apple**            | `apple`           | `APPLE_KEY`           | `APPLE_SECRET`           | -                              | Includes name and email scopes                  |
| **Auth0**            | `auth0`           | `AUTH0_KEY`           | `AUTH0_SECRET`           | `AUTH0_DOMAIN`                 | Domain is required for Auth0                    |
| **Azure AD**         | `azuread`         | `AZUREAD_KEY`         | `AZUREAD_SECRET`         | -                              | Microsoft Azure Active Directory                |
| **Battle.net**       | `battlenet`       | `BATTLENET_KEY`       | `BATTLENET_SECRET`       | -                              | Blizzard Entertainment                          |
| **Bitbucket**        | `bitbucket`       | `BITBUCKET_KEY`       | `BITBUCKET_SECRET`       | -                              | Atlassian Bitbucket                             |
| **Box**              | `box`             | `BOX_KEY`             | `BOX_SECRET`             | -                              | Box cloud storage                               |
| **Dailymotion**      | `dailymotion`     | `DAILYMOTION_KEY`     | `DAILYMOTION_SECRET`     | -                              | Includes email scope                            |
| **Deezer**           | `deezer`          | `DEEZER_KEY`          | `DEEZER_SECRET`          | -                              | Includes email scope                            |
| **DigitalOcean**     | `digitalocean`    | `DIGITALOCEAN_KEY`    | `DIGITALOCEAN_SECRET`    | -                              | Includes read scope                             |
| **Discord**          | `discord`         | `DISCORD_KEY`         | `DISCORD_SECRET`         | -                              | Includes identify and email scopes              |
| **Dropbox**          | `dropbox`         | `DROPBOX_KEY`         | `DROPBOX_SECRET`         | -                              | -                                               |
| **EVE Online**       | `eveonline`       | `EVEONLINE_KEY`       | `EVEONLINE_SECRET`       | -                              | CCP Games                                       |
| **Facebook**         | `facebook`        | `FACEBOOK_KEY`        | `FACEBOOK_SECRET`        | -                              | Includes email and public_profile scopes        |
| **Fitbit**           | `fitbit`          | `FITBIT_KEY`          | `FITBIT_SECRET`          | -                              | -                                               |
| **Gitea**            | `gitea`           | `GITEA_KEY`           | `GITEA_SECRET`           | -                              | Self-hosted Git service                         |
| **GitHub**           | `github`          | `GITHUB_KEY`          | `GITHUB_SECRET`          | -                              | Includes read:user and user:email scopes        |
| **GitLab**           | `gitlab`          | `GITLAB_KEY`          | `GITLAB_SECRET`          | -                              | -                                               |
| **Google**           | `google`          | `GOOGLE_KEY`          | `GOOGLE_SECRET`          | -                              | -                                               |
| **Heroku**           | `heroku`          | `HEROKU_KEY`          | `HEROKU_SECRET`          | -                              | -                                               |
| **Instagram**        | `instagram`       | `INSTAGRAM_KEY`       | `INSTAGRAM_SECRET`       | -                              | -                                               |
| **Intercom**         | `intercom`        | `INTERCOM_KEY`        | `INTERCOM_SECRET`        | -                              | -                                               |
| **Kakao**            | `kakao`           | `KAKAO_KEY`           | `KAKAO_SECRET`           | -                              | Korean social platform                          |
| **Last.fm**          | `lastfm`          | `LASTFM_KEY`          | `LASTFM_SECRET`          | -                              | Music platform                                  |
| **LINE**             | `line`            | `LINE_KEY`            | `LINE_SECRET`            | -                              | Includes profile, openid, and email scopes      |
| **LinkedIn**         | `linkedin`        | `LINKEDIN_KEY`        | `LINKEDIN_SECRET`        | -                              | -                                               |
| **Mastodon**         | `mastodon`        | `MASTODON_KEY`        | `MASTODON_SECRET`        | -                              | Includes read:accounts scope                    |
| **Meetup**           | `meetup`          | `MEETUP_KEY`          | `MEETUP_SECRET`          | -                              | -                                               |
| **Microsoft Online** | `microsoftonline` | `MICROSOFTONLINE_KEY` | `MICROSOFTONLINE_SECRET` | -                              | Microsoft 365                                   |
| **Naver**            | `naver`           | `NAVER_KEY`           | `NAVER_SECRET`           | -                              | Korean search engine                            |
| **Nextcloud**        | `nextcloud`       | `NEXTCLOUD_KEY`       | `NEXTCLOUD_SECRET`       | `NEXTCLOUD_URL`                | Self-hosted cloud platform                      |
| **Okta**             | `okta`            | `OKTA_ID`             | `OKTA_SECRET`            | `OKTA_ORG_URL`                 | Enterprise identity platform                    |
| **OneDrive**         | `onedrive`        | `ONEDRIVE_KEY`        | `ONEDRIVE_SECRET`        | -                              | Microsoft OneDrive                              |
| **OpenID Connect**   | `openid-connect`  | `OPENID_CONNECT_KEY`  | `OPENID_CONNECT_SECRET`  | `OPENID_CONNECT_DISCOVERY_URL` | Generic OpenID Connect provider                 |
| **Patreon**          | `patreon`         | `PATREON_KEY`         | `PATREON_SECRET`         | -                              | Creator funding platform                        |
| **PayPal**           | `paypal`          | `PAYPAL_KEY`          | `PAYPAL_SECRET`          | `PAYPAL_ENV` (optional)        | Set PAYPAL_ENV=sandbox for testing              |
| **Salesforce**       | `salesforce`      | `SALESFORCE_KEY`      | `SALESFORCE_SECRET`      | -                              | CRM platform                                    |
| **Seatalk**          | `seatalk`         | `SEATALK_KEY`         | `SEATALK_SECRET`         | -                              | -                                               |
| **Shopify**          | `shopify`         | `SHOPIFY_KEY`         | `SHOPIFY_SECRET`         | -                              | Includes read customers and orders scopes       |
| **Slack**            | `slack`           | `SLACK_KEY`           | `SLACK_SECRET`           | -                              | -                                               |
| **SoundCloud**       | `soundcloud`      | `SOUNDCLOUD_KEY`      | `SOUNDCLOUD_SECRET`      | -                              | -                                               |
| **Spotify**          | `spotify`         | `SPOTIFY_KEY`         | `SPOTIFY_SECRET`         | -                              | -                                               |
| **Steam**            | `steam`           | `STEAM_KEY`           | -                        | -                              | Only requires API key, no secret                |
| **Strava**           | `strava`          | `STRAVA_KEY`          | `STRAVA_SECRET`          | -                              | Fitness tracking platform                       |
| **Stripe**           | `stripe`          | `STRIPE_KEY`          | `STRIPE_SECRET`          | -                              | Payment processing                              |
| **TikTok**           | `tiktok`          | `TIKTOK_KEY`          | `TIKTOK_SECRET`          | -                              | -                                               |
| **Twitch**           | `twitch`          | `TWITCH_KEY`          | `TWITCH_SECRET`          | -                              | -                                               |
| **Twitter v2**       | `twitterv2`       | `TWITTER_KEY`         | `TWITTER_SECRET`         | -                              | Uses Twitter API v2 (Essential tier compatible) |
| **Typetalk**         | `typetalk`        | `TYPETALK_KEY`        | `TYPETALK_SECRET`        | -                              | Includes "my" scope                             |
| **Uber**             | `uber`            | `UBER_KEY`            | `UBER_SECRET`            | -                              | -                                               |
| **VK**               | `vk`              | `VK_KEY`              | `VK_SECRET`              | -                              | Russian social network                          |
| **WeCom**            | `wecom`           | `WECOM_CORP_ID`       | `WECOM_SECRET`           | `WECOM_AGENT_ID`               | WeChat Work (enterprise)                        |
| **WePay**            | `wepay`           | `WEPAY_KEY`           | `WEPAY_SECRET`           | -                              | Includes view_user scope                        |
| **Xero**             | `xero`            | `XERO_KEY`            | `XERO_SECRET`            | -                              | Accounting software                             |
| **Yahoo**            | `yahoo`           | `YAHOO_KEY`           | `YAHOO_SECRET`           | -                              | ⚠️ Uses hardcoded HTTPS callback                |
| **Yammer**           | `yammer`          | `YAMMER_KEY`          | `YAMMER_SECRET`          | -                              | Microsoft Yammer                                |
| **Yandex**           | `yandex`          | `YANDEX_KEY`          | `YANDEX_SECRET`          | -                              | Russian search engine                           |
| **Zoom**             | `zoom`            | `ZOOM_KEY`            | `ZOOM_SECRET`            | -                              | Includes read:user scope                        |

## Special Configuration Notes

### Yahoo

Yahoo has special requirements and uses a hardcoded callback URL of `https://localhost.com`. You need to configure your
Yahoo app to use this specific callback URL.

### Auth0

Auth0 requires a domain configuration. Make sure to set the `AUTH0_DOMAIN` environment variable to your Auth0 domain.

### Okta

Okta requires both the organization URL (`OKTA_ORG_URL`) and uses `OKTA_ID` instead of `OKTA_KEY`.

### Nextcloud

Nextcloud requires the `NEXTCLOUD_URL` to specify your Nextcloud instance URL.

### WeCom (WeChat Work)

WeCom requires three variables: `WECOM_CORP_ID`, `WECOM_SECRET`, and `WECOM_AGENT_ID`.

### PayPal

PayPal uses production URLs by default. For testing, set `PAYPAL_ENV=sandbox`.

### Steam

Steam only requires the `STEAM_KEY` (API key) and does not use a secret.

### Twitter

The implementation uses Twitter API v2 (`twitterv2`) which is compatible with the Essential API tier. There's also
support for authentication mode instead of authorization (commented out in code).

### OpenID Connect

OpenID Connect is a generic provider that supports any OpenID Connect compliant service. It requires the
`OPENID_CONNECT_DISCOVERY_URL` which should point to the OpenID Connect Auto Discovery URL (as
per https://openid.net/specs/openid-connect-discovery-1_0-17.html).

## Usage

1. Choose the OAuth providers you want to support
2. Register your application with each provider to get the required credentials
3. Set the environment variables for each provider
4. The providers will be automatically enabled when their primary key is detected

## Example Configuration

```bash
# Google OAuth
export GOOGLE_KEY="your-google-client-id"
export GOOGLE_SECRET="your-google-client-secret"

# GitHub OAuth
export GITHUB_KEY="your-github-client-id"
export GITHUB_SECRET="your-github-client-secret"

# Auth0 OAuth (requires domain)
export AUTH0_KEY="your-auth0-client-id"
export AUTH0_SECRET="your-auth0-client-secret"
export AUTH0_DOMAIN="your-tenant.auth0.com"
```

## Authentication Endpoints

Once configured, each provider will be available at:

- **Login**: `/auth/{provider}` (e.g., `/auth/google`)
- **Callback**: `/auth/{provider}/callback` (e.g., `/auth/google/callback`)
- **Logout**: `/auth/{provider}/logout` (e.g., `/auth/google/logout`)

## Testing with Mock Providers

For testing purposes, you can use mock OAuth providers by setting:

```bash
export OAUTH_MOCK_SERVER_URL="http://localhost:8080"
```

When this is set, the system will use mock providers instead of real OAuth endpoints. This requires building with the
test tag: `go test -tags=test`.
