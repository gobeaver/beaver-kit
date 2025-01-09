# Captcha Package

A unified interface for various CAPTCHA services supporting Google reCAPTCHA (v2/v3), hCaptcha, and Cloudflare Turnstile.

## Purpose and Features

- **Multiple Providers**: Support for Google reCAPTCHA, hCaptcha, and Cloudflare Turnstile
- **Unified Interface**: Single API for all captcha providers
- **Zero-Config**: Works with sensible defaults
- **Environment Configuration**: Configure via environment variables
- **Validation**: Server-side token validation with context support
- **HTML Generation**: Automatic generation of client-side HTML/JavaScript
- **Disabled Mode**: Built-in support for development/testing with captcha disabled

## Installation

```bash
go get github.com/gobeaver/beaver-kit/captcha
```

## Usage Examples

### Zero-Config Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/gobeaver/beaver-kit/captcha"
)

func main() {
    // Initialize with environment configuration
    if err := captcha.Init(); err != nil {
        panic(err)
    }
    
    // Get the global instance
    service := captcha.Service()
    
    // Generate HTML for the form
    html := service.GenerateHTML()
    fmt.Println(html)
    
    // Validate a token
    token := "user-submitted-token"
    remoteIP := "192.168.1.1"
    valid, err := service.Validate(context.Background(), token, remoteIP)
    if err != nil {
        fmt.Printf("Validation error: %v\n", err)
    }
    fmt.Printf("Token valid: %v\n", valid)
}
```

### Environment-Based Configuration

Set environment variables:

```bash
export BEAVER_CAPTCHA_ENABLED=true
export BEAVER_CAPTCHA_PROVIDER=recaptcha
export BEAVER_CAPTCHA_SITE_KEY=your-site-key
export BEAVER_CAPTCHA_SECRET_KEY=your-secret-key
export BEAVER_CAPTCHA_VERSION=2
```

Then use in code:

```go
// Automatically loads from environment
if err := captcha.Init(); err != nil {
    panic(err)
}

service := captcha.Service()
```

### Direct Configuration

```go
// Create with specific configuration
cfg := captcha.Config{
    Provider:  "hcaptcha",
    SiteKey:   "your-hcaptcha-site-key",
    SecretKey: "your-hcaptcha-secret-key",
    Enabled:   true,
}

// Option 1: Initialize global instance with config
if err := captcha.Init(cfg); err != nil {
    panic(err)
}

// Option 2: Create a new instance
service, err := captcha.New(cfg)
if err != nil {
    panic(err)
}
```

### Using Specific Implementations

```go
// Create Google reCAPTCHA v2
googleV2 := captcha.NewGoogleCaptcha("site-key", "secret-key", 2)

// Create Google reCAPTCHA v3
googleV3 := captcha.NewGoogleCaptcha("site-key", "secret-key", 3)

// Create hCaptcha
hcaptcha := captcha.NewHCaptcha("site-key", "secret-key")

// Create Cloudflare Turnstile
turnstile := captcha.NewTurnstile("site-key", "secret-key")
```

### Disabled Mode (Development/Testing)

```go
cfg := captcha.Config{
    Enabled: false, // Captcha validation always returns true
}

service, _ := captcha.New(cfg)
valid, _ := service.Validate(context.Background(), "any-token", "any-ip")
// valid is always true when disabled
```

### Testing with Reset

```go
func TestCaptchaValidation(t *testing.T) {
    // Clean up after test
    defer captcha.Reset()
    
    // Initialize with test configuration
    testConfig := captcha.Config{
        Provider:  "recaptcha",
        SiteKey:   "test-site-key",
        SecretKey: "test-secret-key",
        Enabled:   true,
    }
    
    if err := captcha.Init(testConfig); err != nil {
        t.Fatal(err)
    }
    
    service := captcha.Service()
    // ... run tests
}
```

## Configuration Options

| Field | Environment Variable | Default | Description |
|-------|---------------------|---------|-------------|
| Provider | BEAVER_CAPTCHA_PROVIDER | recaptcha | Captcha service provider (recaptcha, hcaptcha, turnstile) |
| SiteKey | BEAVER_CAPTCHA_SITE_KEY | - | Public key for the captcha service |
| SecretKey | BEAVER_CAPTCHA_SECRET_KEY | - | Private key for server-side validation |
| Version | BEAVER_CAPTCHA_VERSION | 2 | Captcha version (only for recaptcha: 2 or 3) |
| Enabled | BEAVER_CAPTCHA_ENABLED | false | Whether captcha validation is active |

## Provider-Specific Features

### Google reCAPTCHA

- **v2**: Traditional "I'm not a robot" checkbox
- **v3**: Invisible captcha with score-based validation

```go
// reCAPTCHA v3 with score validation
googleService := captcha.NewGoogleCaptcha(siteKey, secretKey, 3)
if v3Service, ok := googleService.(*captcha.GoogleCaptchaService); ok {
    valid, score, err := v3Service.ValidateV3WithScore(
        ctx, token, remoteIP, "login", 0.5,
    )
}
```

### hCaptcha

- Privacy-focused alternative to reCAPTCHA
- Supports enterprise features (score, reasons)

### Cloudflare Turnstile

- Cloudflare's privacy-preserving captcha
- No user interaction required in most cases

## HTML Generation

Each provider generates appropriate HTML:

```go
// reCAPTCHA v2
<script src="https://www.google.com/recaptcha/api.js" async defer></script>
<div class="g-recaptcha" data-sitekey="your-site-key"></div>

// hCaptcha
<script src="https://js.hcaptcha.com/1/api.js" async defer></script>
<div class="h-captcha" data-sitekey="your-site-key"></div>

// Turnstile
<script src="https://challenges.cloudflare.com/turnstile/v0/api.js" async defer></script>
<div class="cf-turnstile" data-sitekey="your-site-key"></div>
```

## Error Handling

The package defines specific errors for better debugging:

```go
var (
    ErrInvalidConfig    = errors.New("invalid configuration")
    ErrNotInitialized   = errors.New("service not initialized")
    ErrProviderRequired = errors.New("captcha provider required")
    ErrInvalidProvider  = errors.New("invalid captcha provider")
    ErrKeysRequired     = errors.New("site key and secret key required")
)
```

Example error handling:

```go
service, err := captcha.New(cfg)
if err != nil {
    if errors.Is(err, captcha.ErrInvalidProvider) {
        // Handle invalid provider
    } else if errors.Is(err, captcha.ErrKeysRequired) {
        // Handle missing keys
    }
}
```

## Complete Example

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "net/http"
    
    "github.com/gobeaver/beaver-kit/captcha"
)

func main() {
    // Initialize captcha service
    if err := captcha.Init(); err != nil {
        log.Fatal(err)
    }
    
    http.HandleFunc("/", formHandler)
    http.HandleFunc("/submit", submitHandler)
    
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func formHandler(w http.ResponseWriter, r *http.Request) {
    service := captcha.Service()
    html := fmt.Sprintf(`
        <html>
        <head><title>Captcha Example</title></head>
        <body>
            <form action="/submit" method="POST">
                <input type="text" name="username" placeholder="Username" required>
                %s
                <button type="submit">Submit</button>
            </form>
        </body>
        </html>
    `, service.GenerateHTML())
    
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}

func submitHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    // Get the captcha response token
    token := r.FormValue("g-recaptcha-response") // For reCAPTCHA
    // token := r.FormValue("h-captcha-response") // For hCaptcha
    // token := r.FormValue("cf-turnstile-response") // For Turnstile
    
    // Validate the token
    service := captcha.Service()
    valid, err := service.Validate(context.Background(), token, r.RemoteAddr)
    
    if err != nil {
        log.Printf("Captcha validation error: %v", err)
        http.Error(w, "Captcha validation failed", http.StatusBadRequest)
        return
    }
    
    if !valid {
        http.Error(w, "Invalid captcha", http.StatusBadRequest)
        return
    }
    
    // Process the form...
    username := r.FormValue("username")
    fmt.Fprintf(w, "Welcome, %s! Captcha validated successfully.", username)
}
```

## License

Part of the Beaver Kit project. See the main LICENSE file for details.