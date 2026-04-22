# goingcrypt — API Documentation

This directory contains the [Bruno](https://www.usebruno.com/) collection for the **goingcrypt** REST API.

## Getting started

1. [Download Bruno](https://www.usebruno.com/downloads) (free, open-source)
2. Open Bruno → **Open Collection** → select this `bruno/` folder
3. Select the **local** environment (top-right dropdown)
4. Start making requests

## Environment variables

| Variable        | Default                    | Description                              |
|-----------------|----------------------------|------------------------------------------|
| `base_url`      | `http://localhost:8080`    | API base URL                             |
| `access_token`  | _(empty)_                  | Auto-populated after Sign Up / Login     |
| `refresh_token` | _(empty)_                  | Auto-populated after Sign Up / Login     |
| `mfa_secret`    | _(empty)_                  | Auto-populated after Enable MFA          |
| `link_slug`     | _(empty)_                  | Auto-populated after Create Link         |

## Endpoints

### Auth (`/api/v1/auth`)

| # | Name              | Method | Path               | Auth     |
|---|-------------------|--------|--------------------|----------|
| 1 | Sign Up           | POST   | `/signup`          | Public   |
| 2 | Login             | POST   | `/login`           | Public   |
| 3 | Login with MFA    | POST   | `/login`           | Public   |
| 4 | Refresh Tokens    | POST   | `/refresh`         | Public   |
| 5 | Logout            | POST   | `/logout`          | Public   |
| 6 | Enable MFA        | POST   | `/mfa/enable`      | Bearer   |
| 7 | Confirm MFA       | POST   | `/mfa/confirm`     | Bearer   |

### Links (`/api/v1/links`)

| # | Name         | Method | Path        | Auth            |
|---|--------------|--------|-------------|-----------------|
| 1 | Create Link  | POST   | `/`         | Optional Bearer |
| 2 | Access Link  | GET    | `/{slug}`   | Public          |
| 3 | Delete Link  | DELETE | `/{slug}`   | Bearer          |

## MFA enrollment flow

```
1. Sign up or log in  →  save access_token
2. POST /mfa/enable   →  scan provisioning_uri QR in Google Authenticator
3. POST /mfa/confirm  →  { secret, code: "<6-digit from app>" }  →  MFA active

Next login:
4. POST /login        →  { username, password }  →  { mfa_required: true }
5. POST /login        →  { username, password, mfa_code: "123456" }  →  tokens
```

## Encrypted link flow

```
1. Client encrypts payload with a passphrase  →  ciphered_text
2. POST /links         →  { key, ciphered_text, expires_in? }  →  { slug }
3. Share: <base_url>/api/v1/links/<slug> + passphrase (out of band)
4. GET  /links/:slug   →  { key }  →  { ciphered_text }
5. Client decrypts ciphered_text with passphrase  →  original payload
```
