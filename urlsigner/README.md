# URL Signer for Beaver Kit

A component of the [Beaver Kit](https://github.com/gobeaver/beaver-kit) framework for Go.

A Go package that provides secure URL signing and verification capabilities for implementing temporary access to resources, secure file downloads, and signed API endpoints in your microservice architecture.

## Features

- **Secure Signing**: Generate cryptographically secure signatures for URLs
- **Configurable Expiration**: Set custom expiration times for temporary access
- **Payload Support**: Embed additional data in signed URLs (permissions, metadata, etc.)
- **Verification**: Validate signatures and check expiration before granting access
- **Simple API**: Clean, intuitive interface that's easy to integrate
- **Customizable**: Configure parameter names, algorithms, and default settings
- **Secure by Default**: Uses HMAC-SHA256 with safe defaults
- **Zero External Dependencies**: Pure Go implementation that relies only on the standard library
- **Environment Variable Configuration**: Full support for environment-based configuration following Beaver Kit conventions
- **Global Instance Management**: Thread-safe singleton pattern with easy initialization
- **Comprehensive Error Handling**: Detailed error types for different failure scenarios

## Installation

```bash
go get github.com/gobeaver/beaver-kit
```

## Configuration

The URL Signer follows the Beaver Kit conventions and can be configured using environment variables:

| Environment Variable                | Description                                 | Default   |
|-------------------------------------|---------------------------------------------|-----------|
| `BEAVER_URLSIGNER_SECRET_KEY`       | HMAC secret key for signing URLs            | Required  |
| `BEAVER_URLSIGNER_DEFAULT_EXPIRY`   | Default expiration duration for signed URLs | `30m`     |
| `BEAVER_URLSIGNER_ALGORITHM`        | Hashing algorithm to use                    | `sha256`  |
| `BEAVER_URLSIGNER_SIGNATURE_PARAM`  | Query parameter name for signature          | `sig`     |
| `BEAVER_URLSIGNER_EXPIRES_PARAM`    | Query parameter name for expiration         | `expires` |
| `BEAVER_URLSIGNER_PAYLOAD_PARAM`    | Query parameter name for payload            | `payload` |

### Setting Environment Variables

You can set environment variables in several ways:

**Using a .env file:**
```env
BEAVER_URLSIGNER_SECRET_KEY=your-super-secret-key-here
BEAVER_URLSIGNER_DEFAULT_EXPIRY=1h
BEAVER_URLSIGNER_ALGORITHM=sha256
BEAVER_URLSIGNER_SIGNATURE_PARAM=signature
BEAVER_URLSIGNER_EXPIRES_PARAM=exp
BEAVER_URLSIGNER_PAYLOAD_PARAM=data
```

**Using shell export:**
```bash
export BEAVER_URLSIGNER_SECRET_KEY="your-super-secret-key-here"
export BEAVER_URLSIGNER_DEFAULT_EXPIRY="1h"
```

**Using Docker:**
```bash
docker run -e BEAVER_URLSIGNER_SECRET_KEY="your-key" your-app
```

## Quick Start

### Zero-Configuration (Using Environment Variables)

```go
import (
    "fmt"
    "time"
    
    "github.com/gobeaver/beaver-kit/urlsigner"
)

func main() {
    // Initialize with environment variables
    // Set BEAVER_URLSIGNER_SECRET_KEY environment variable
    if err := urlsigner.Init(); err != nil {
        panic(err)
    }
    
    // Get the global service instance
    signer := urlsigner.Service()
    
    // Sign a URL with a 30-minute expiration
    signedURL, err := signer.SignURL(
        "https://example.com/download/file.pdf", 
        30*time.Minute, 
        "",
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Signed URL:", signedURL)
    
    // Later, verify the signed URL
    valid, payload, err := signer.VerifyURL(signedURL)
    if err != nil {
        fmt.Printf("URL verification error: %v\n", err)
        return
    }
    
    if valid {
        fmt.Println("URL is valid!")
    } else {
        fmt.Println("URL is invalid or expired")
    }
}
```

### Direct Configuration

```go
import (
    "fmt"
    "time"
    
    "github.com/gobeaver/beaver-kit/urlsigner"
)

func main() {
    // Create a new signer with a secret key
    signer := urlsigner.NewSigner("your-secret-key")
    
    // Sign a URL with a 30-minute expiration
    signedURL, err := signer.SignURL(
        "https://example.com/download/file.pdf", 
        30*time.Minute, 
        "",
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println("Signed URL:", signedURL)
}

## Detailed Usage

### Creating a Signer

#### Using Environment Variables (Recommended)

```go
// Initialize with environment variables
if err := urlsigner.Init(); err != nil {
    log.Fatal(err)
}

// Get the global service instance
signer := urlsigner.Service()
```

#### Using Configuration Struct

```go
// Create with specific configuration
config := urlsigner.Config{
    SecretKey:      "your-secret-key",
    DefaultExpiry:  1 * time.Hour,
    Algorithm:      "sha256",
    SignatureParam: "s",
    ExpiresParam:   "e",
    PayloadParam:   "p",
}

signer, err := urlsigner.New(config)
if err != nil {
    log.Fatal(err)
}
```

#### Direct Creation (Legacy API)

```go
// Create with just a secret key (uses default options)
signer := urlsigner.NewSigner("your-secret-key")

// Or with custom options
options := urlsigner.SignerOptions{
    SecretKey:     "your-secret-key",
    DefaultExpiry: 1 * time.Hour,
    Algorithm:     "sha256",
    QueryParams: &urlsigner.SignatureParams{
        Signature: "s",     // Short parameter names
        Expires:   "e",
        Payload:   "p",
    },
}

signer := urlsigner.NewSignerWithOptions(options)
```

## Error Handling

The URL Signer provides specific error types for different scenarios:

```go
// Package-specific errors
var (
    ErrInvalidConfig      = errors.New("invalid configuration")
    ErrNotInitialized     = errors.New("service not initialized")
    ErrInvalidURL         = errors.New("invalid URL")
    ErrSignatureNotFound  = errors.New("signature not found")
    ErrExpirationNotFound = errors.New("expiration not found")
    ErrExpired            = errors.New("URL has expired")
    ErrInvalidSignature   = errors.New("invalid signature")
)
```

### Example Error Handling

```go
valid, payload, err := signer.VerifyURL(signedURL)
if err != nil {
    switch err {
    case urlsigner.ErrExpired:
        fmt.Println("URL has expired")
    case urlsigner.ErrInvalidSignature:
        fmt.Println("URL signature is invalid")
    case urlsigner.ErrSignatureNotFound:
        fmt.Println("URL is not signed")
    default:
        fmt.Printf("Verification error: %v\n", err)
    }
    return
}
```

### Signing URLs

Basic URL signing with default expiration:

```go
// Sign with default expiration time (30 minutes)
signedURL, err := signer.SignURLWithDefaultExpiry("https://example.com/resource/123", "")
```

URL signing with custom expiration:

```go
// Sign with custom expiration (5 minutes)
signedURL, err := signer.SignURL("https://example.com/resource/123", 5*time.Minute, "")
```

### Including Payloads

You can embed additional data in the signed URL:

```go
// Sign a URL with a JSON payload
payload := `{"user_id": 42, "permissions": ["read"]}`
signedURL, err := signer.SignURL(
    "https://example.com/api/resource", 
    15*time.Minute, 
    payload,
)

// Later, extract and use the payload
valid, extractedPayload, err := signer.VerifyURL(signedURL)
if valid {
    fmt.Println("Payload:", extractedPayload)
    // Parse and use the payload...
}
```

### Verifying URLs

Basic verification:

```go
valid, payload, err := signer.VerifyURL(signedURL)
if err != nil {
    fmt.Printf("Verification error: %v\n", err)
    return
}

if valid {
    fmt.Println("URL is valid!")
    if payload != "" {
        fmt.Println("Payload:", payload)
    }
} else {
    fmt.Println("URL is invalid or expired")
}
```

Checking expiration:

```go
// Check if a URL has expired
expired, err := signer.IsExpired(signedURL)
if err != nil {
    fmt.Printf("Error checking expiration: %v\n", err)
    return
}

if expired {
    fmt.Println("URL has expired")
} else {
    // Check remaining validity
    remaining, err := signer.RemainingValidity(signedURL)
    if err != nil {
        fmt.Printf("Error checking validity: %v\n", err)
        return
    }
    
    fmt.Printf("URL is valid for another %v\n", remaining)
}
```

## Integration with Microservices

### File Download Service

```go
func setupFileDownloadServer(secretKey string) {
    signer := urlsigner.NewSigner(secretKey)
    
    // Handler to generate signed download URLs
    http.HandleFunc("/generate-download", func(w http.ResponseWriter, r *http.Request) {
        fileID := r.URL.Query().Get("id")
        if fileID == "" {
            http.Error(w, "Missing file ID", http.StatusBadRequest)
            return
        }
        
        // Generate download URL valid for 15 minutes
        fileURL := fmt.Sprintf("https://%s/download/%s", r.Host, fileID)
        signedURL, err := signer.SignURL(fileURL, 15*time.Minute, "")
        if err != nil {
            http.Error(w, "Failed to generate download URL", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"download_url": "%s"}`, signedURL)
    })
    
    // Handler to serve file downloads
    http.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
        // Verify the signed URL
        valid, _, err := signer.VerifyURL(r.URL.String())
        if err != nil || !valid {
            http.Error(w, "Invalid or expired download link", http.StatusForbidden)
            return
        }
        
        // Extract the file ID from the path
        parts := strings.Split(r.URL.Path, "/")
        fileID := parts[len(parts)-1]
        
        // Serve the file...
        serveFile(w, r, fileID)
    })
}
```

### Image Processing Service

```go
func setupImageProcessingServer(secretKey string) {
    signer := urlsigner.NewSigner(secretKey)
    
    // Handler to generate signed image processing URLs
    http.HandleFunc("/resize", func(w http.ResponseWriter, r *http.Request) {
        // Get image URL and dimensions
        imageURL := r.URL.Query().Get("url")
        width := r.URL.Query().Get("width")
        height := r.URL.Query().Get("height")
        
        if imageURL == "" {
            http.Error(w, "Missing image URL", http.StatusBadRequest)
            return
        }
        
        // Create a payload with processing parameters
        payload := fmt.Sprintf(`{"width": "%s", "height": "%s"}`, width, height)
        
        // Sign the image URL with the payload
        processingURL := fmt.Sprintf("https://%s/process?img=%s", r.Host, url.QueryEscape(imageURL))
        signedURL, err := signer.SignURL(processingURL, 10*time.Minute, payload)
        if err != nil {
            http.Error(w, "Failed to generate processing URL", http.StatusInternalServerError)
            return
        }
        
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"processing_url": "%s"}`, signedURL)
    })
    
    // Handler to process images
    http.HandleFunc("/process", func(w http.ResponseWriter, r *http.Request) {
        // Verify the signed URL
        valid, payload, err := signer.VerifyURL(r.URL.String())
        if err != nil || !valid {
            http.Error(w, "Invalid or expired processing link", http.StatusForbidden)
            return
        }
        
        // Parse the payload to get processing parameters
        var params map[string]string
        if err := json.Unmarshal([]byte(payload), &params); err != nil {
            http.Error(w, "Invalid parameters", http.StatusBadRequest)
            return
        }
        
        // Get the image URL
        imageURL := r.URL.Query().Get("img")
        
        // Process the image...
        processedImage, err := processImage(imageURL, params["width"], params["height"])
        if err != nil {
            http.Error(w, "Processing failed", http.StatusInternalServerError)
            return
        }
        
        // Serve the processed image
        w.Header().Set("Content-Type", "image/jpeg")
        w.Write(processedImage)
    })
}
```

## Security Considerations

1. **Keep Secret Keys Secure**: The security of signed URLs depends entirely on keeping your secret key private. Store it securely and never expose it in client-side code.

2. **Set Appropriate Expiration Times**: Use the shortest reasonable expiration time for your use case to minimize the window of potential misuse.

3. **Use HTTPS Only**: Always use HTTPS for signed URLs to prevent interception or man-in-the-middle attacks.

4. **Implement Rate Limiting**: Add rate limiting to endpoints that generate or accept signed URLs to prevent abuse.

5. **Consider URL Length**: Be mindful that adding signatures and payloads increases URL length. Some browsers and servers have URL length limitations.

6. **Rotate Secret Keys**: Periodically rotate your secret keys, especially after personnel changes or potential security incidents.

## API Reference

### Types

```go
// Config defines the configuration for URL signer
type Config struct {
    SecretKey      string        // HMAC secret key for signing URLs
    DefaultExpiry  time.Duration // Default expiration duration
    Algorithm      string        // Hashing algorithm (only sha256 supported)
    SignatureParam string        // Query parameter name for signature
    ExpiresParam   string        // Query parameter name for expiration
    PayloadParam   string        // Query parameter name for payload
}

// SignatureParams customizes how signature parameters appear in URLs
type SignatureParams struct {
    Signature string // Query parameter name for signature
    Expires   string // Query parameter name for expiration
    Payload   string // Query parameter name for additional payload
}

// SignerOptions configures the Signer behavior (legacy)
type SignerOptions struct {
    SecretKey     string
    DefaultExpiry time.Duration
    Algorithm     string
    QueryParams   *SignatureParams
}
```

### Initialization and Creation

```go
// Initialize the global instance with optional config
func Init(configs ...Config) error

// Get configuration from environment variables
func GetConfig() (*Config, error)

// Create a new instance with given config
func New(cfg Config) (*Signer, error)

// Get the global service instance
func Service() *Signer

// Reset the global instance (for testing)
func Reset()

// Legacy constructors (kept for backward compatibility)
func NewSigner(secretKey string) *Signer
func NewSignerWithOptions(options SignerOptions) *Signer
```

### Signing URLs

```go
// Sign a URL with custom expiration time and payload
func (s *Signer) SignURL(rawURL string, expiry time.Duration, payload string) (string, error)

// Sign a URL with the default expiration time
func (s *Signer) SignURLWithDefaultExpiry(rawURL string, payload string) (string, error)
```

### Verifying URLs

```go
// Verify a signed URL and return payload if valid
func (s *Signer) VerifyURL(signedURL string) (bool, string, error)

// Check if a signed URL has expired
func (s *Signer) IsExpired(signedURL string) (bool, error)

// Get the expiration time of a signed URL
func (s *Signer) GetExpirationTime(signedURL string) (time.Time, error)

// Get the remaining validity time of a signed URL
func (s *Signer) RemainingValidity(signedURL string) (time.Duration, error)

// Extract payload from a signed URL without verification
func (s *Signer) ExtractPayload(signedURL string) (string, error)
```

## Testing

The package includes comprehensive tests. When testing, use the `Reset()` function to clear the global instance between tests:

```go
func TestMyFunction(t *testing.T) {
    defer urlsigner.Reset() // Clean up after test
    
    config := urlsigner.Config{
        SecretKey:     "test-key",
        DefaultExpiry: 10 * time.Minute,
        Algorithm:     "sha256",
    }
    
    if err := urlsigner.Init(config); err != nil {
        t.Fatal(err)
    }
    
    // Your test code here...
}
```

### Testing Environment Configuration

For integration tests that require environment variables:

```go
func TestWithEnvironment(t *testing.T) {
    // Set test environment variables
    os.Setenv("BEAVER_URLSIGNER_SECRET_KEY", "test-secret-key")
    os.Setenv("BEAVER_URLSIGNER_DEFAULT_EXPIRY", "5m")
    defer func() {
        os.Unsetenv("BEAVER_URLSIGNER_SECRET_KEY")
        os.Unsetenv("BEAVER_URLSIGNER_DEFAULT_EXPIRY")
        urlsigner.Reset()
    }()
    
    // Initialize from environment
    if err := urlsigner.Init(); err != nil {
        t.Fatal(err)
    }
    
    signer := urlsigner.Service()
    // Test with the configured signer...
}
```

## Debugging

### Configuration Debugging

Enable configuration debugging to see what values are being loaded:

```bash
BEAVER_CONFIG_DEBUG=true ./your-app
```

This will print all loaded configuration values:
```
[BEAVER] BEAVER_URLSIGNER_SECRET_KEY=your-secret-key
[BEAVER] BEAVER_URLSIGNER_DEFAULT_EXPIRY=30m
[BEAVER] BEAVER_URLSIGNER_ALGORITHM=sha256
```

### Common Issues

**1. "service not initialized" error:**
```go
// Make sure to call Init() before using Service()
if err := urlsigner.Init(); err != nil {
    log.Fatal(err)
}
signer := urlsigner.Service()
```

**2. "invalid configuration" error:**
```go
// Check that required environment variables are set
cfg, err := urlsigner.GetConfig()
if err != nil {
    log.Printf("Config error: %v", err)
    // Check BEAVER_URLSIGNER_SECRET_KEY is set
}
```

## Best Practices

1. **Use Environment Variables for Production**: Set configuration via environment variables rather than hardcoding values in your application.

2. **Keep Secret Keys Secret**: Never commit secret keys to version control. Use environment variables or secure secret management systems.

3. **Use Appropriate Expiration Times**: Balance security and usability by setting expiration times appropriate for your use case.

4. **Test with Different Configurations**: Use the `Reset()` function in tests to ensure clean state between different configuration scenarios.

5. **Handle Errors Gracefully**: Always check for and handle errors when verifying URLs, and provide meaningful error messages to users.

6. **Monitor URL Usage**: Consider logging URL generation and verification events for security monitoring.

## License

This package is available under the Apache 2.0 License. See the LICENSE file in the repository for more information.