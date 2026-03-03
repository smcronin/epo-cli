# Authentication

OPS uses OAuth 2.0 Client Credentials flow. No user login — just a consumer key and secret.

## Registration

1. Go to https://developers.epo.org
2. Register an account (name, email, address, phone)
3. Once approved, go to **My Apps** → **Add a new App**
4. Select **OPS v3.2** as the target system
5. Copy the **Consumer Key** and **Consumer Secret**

> Note: Your OPS login is completely separate from EPO Forum, Espacenet, or any other EPO system. Do NOT reuse passwords.

---

## Getting a Token

Tokens are valid for ~20 minutes. Request a new one when you get a `invalid_access_token` error.

### Step 1 — Base64 encode your credentials

```
base64("consumer_key:consumer_secret")
```

### Step 2 — POST to the token endpoint

```bash
curl -X POST https://ops.epo.org/3.2/auth/accesstoken \
  -H "Authorization: Basic $(echo -n 'YOUR_KEY:YOUR_SECRET' | base64)" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=client_credentials"
```

**Response:**
```json
{
  "issued_at": "1364247843353",
  "application_name": "511d82a3-aa0e-4775-ba48-05ccd9275c56",
  "scope": "core",
  "status": "approved",
  "expires_in": "1199",
  "api_product_list": "[ops-prod]",
  "token_type": "Bearer",
  "access_token": "4AWoepfVNgf09DRmimEnGdXcgoFU",
  "organization_name": "epo",
  "refresh_count": "0"
}
```

Token is valid for ~1199 seconds (20 minutes).

### Step 3 — Use the token

```bash
curl https://ops.epo.org/rest-services/published-data/publication/epodoc/EP1000000/biblio \
  -H "Authorization: Bearer 4AWoepfVNgf09DRmimEnGdXcgoFU"
```

---

## Python Example (complete)

```python
import requests
import base64

def get_token(client_id: str, client_secret: str) -> str:
    url = 'https://ops.epo.org/3.2/auth/accesstoken'
    creds = base64.b64encode(f"{client_id}:{client_secret}".encode()).decode()
    headers = {
        'Authorization': f'Basic {creds}',
        'Content-Type': 'application/x-www-form-urlencoded'
    }
    r = requests.post(url, headers=headers, data={'grant_type': 'client_credentials'})
    r.raise_for_status()
    return r.json()['access_token']


def ops_get(path: str, token: str, accept: str = 'application/json') -> dict:
    url = f'https://ops.epo.org/rest-services/{path}'
    headers = {
        'Authorization': f'Bearer {token}',
        'Accept': accept
    }
    r = requests.get(url, headers=headers)
    r.raise_for_status()
    return r.json()


# Usage
token = get_token('YOUR_CLIENT_ID', 'YOUR_CLIENT_SECRET')
data = ops_get('published-data/publication/epodoc/EP1000000/biblio', token)
```

---

## Go Example

```go
package main

import (
    "encoding/base64"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
)

type TokenResponse struct {
    AccessToken string `json:"access_token"`
    ExpiresIn   string `json:"expires_in"`
    TokenType   string `json:"token_type"`
}

func GetToken(clientID, clientSecret string) (string, error) {
    creds := base64.StdEncoding.EncodeToString(
        []byte(fmt.Sprintf("%s:%s", clientID, clientSecret)),
    )

    data := url.Values{}
    data.Set("grant_type", "client_credentials")

    req, _ := http.NewRequest("POST",
        "https://ops.epo.org/3.2/auth/accesstoken",
        strings.NewReader(data.Encode()))

    req.Header.Set("Authorization", "Basic "+creds)
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    var tok TokenResponse
    json.NewDecoder(resp.Body).Decode(&tok)
    return tok.AccessToken, nil
}
```

---

## Error Conditions

| HTTP | Error | Meaning |
|------|-------|---------|
| 400 | `invalid_request` | Missing or malformed param |
| 400 | `invalid_client` | Bad credentials or account blocked |
| 400 | `unsupported_grant_type` | Must be `client_credentials` |
| 400 | `invalid_access_token` | Token expired or invalid |
| 401 | `Client identifier is required` | Base64 encoding malformed |
| 403 | Fair use violation | Quota exceeded |
| 403 | Developer account blocked | Contact EPO support |

---

## Developer Portal

Interactive API explorer at https://developers.epo.org/ops-v32

To use it:
1. Log in
2. Go to a service endpoint, e.g. `GET /published-data/search`
3. My Apps → Add App (if you haven't)
4. Set OAuth 2.0, click Authorize
5. Click "Try Out" → fill params → Execute
