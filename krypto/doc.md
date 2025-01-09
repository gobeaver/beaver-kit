# gofac

## **Package Documentation: krypto**

The **`krypto`** package provides cryptographic functionalities for secure password hashing, token generation, and hashing algorithms.

### **Functions**

### **Argon2idHashPassword**

```go
func Argon2idHashPassword(password string) (string, error)
```

- **Description**: Generates an Argon2id hash from the provided password using a cryptographically secure random salt and configurable parameters.
- **Parameters**:
    - **`password`** (string): The plaintext password to be hashed.
- **Returns**:
    - **`string`**: The hashed password encoded in a string format containing hash parameters.
    - **`error`**: An error encountered during the hashing process, if any.

### **Argon2idVerifyPassword**

```go
func Argon2idVerifyPassword(password, encodedHash string) (bool, error)
```

- **Description**: Verifies a password against an Argon2id hash. It parses the encoded hash string to extract the parameters and salt used to create the original hash, then applies the same parameters to the input password to verify if it matches the stored hash.
- **Parameters**:
    - **`password`** (string): The plaintext password to verify.
    - **`encodedHash`** (string): The encoded hash string to verify against, as returned by Argon2idHashPassword.
- **Returns**:
    - **`bool`**: True if the password matches the hash, false otherwise.
    - **`error`**: An error if the hash format is invalid or parameters couldn't be parsed.

### **BcryptHashPassword**

```go
func BcryptHashPassword(password string) (string, error)
```

- **Description**: Generates a bcrypt hash from the provided password using a cost factor of 12.
- **Parameters**:
    - **`password`** (string): The plaintext password to be hashed.
- **Returns**:
    - **`string`**: The hashed password.
    - **`error`**: An error encountered during the hashing process, if any.

### **BcryptCheckPasswordHash**

```go
func BcryptCheckPasswordHash(password, hash string) bool
```

- **Description**: Compares a bcrypt hashed password with a plaintext password to check if they match.
- **Parameters**:
    - **`password`** (string): The plaintext password to be checked.
    - **`hash`** (string): The bcrypt hashed password to be compared with the plaintext password.
- **Returns**:
    - **`bool`**: A boolean indicating whether the plaintext password matches the bcrypt hash.

### **GenerateHS256Key**

```go
func generateHS256Key() (string, error)
```

- **Description**: Generates a random key suitable for use with the HS256 algorithm.
- **Returns**:
    - **`string`**: The generated key as a base64-encoded string.
    - **`error`**: An error encountered during the key generation process, if any.

### **NewHs256AccessToken**

```go
func NewHs256AccessToken(claims UserClaims) (string, error)
```

- **Description**: Generates an HS256 access token based on the provided user claims.
- **Parameters**:
    - **`claims`** (UserClaims): Claims to be included in the token.
- **Returns**:
    - **`string`**: The generated access token.
    - **`error`**: An error encountered during the token generation process, if any.

### **NewHs256RefreshToken**

```go
func NewHs256RefreshToken(claims jwt.StandardClaims) (string, error)
```

- **Description**: Generates an HS256 refresh token based on the provided claims.
- **Parameters**:
    - **`claims`** (jwt.StandardClaims): Claims to be included in the token.
- **Returns**:
    - **`string`**: The generated refresh token.
    - **`error`**: An error encountered during the token generation process, if any.

### **ParseHs256AccessToken**

```go
func ParseHs256AccessToken(accessToken string) (*UserClaims, error)
```

- **Description**: Parses an HS256 access token and validates its authenticity.
- **Parameters**:
    - **`accessToken`** (string): The access token to be parsed.
- **Returns**:
    - **`UserClaims`**: The parsed user claims if the token is valid.
    - **`error`**: An error encountered during the parsing process, if any.

### **ParseHs256RefreshToken**

```go
func ParseHs256RefreshToken(refreshToken string) *jwt.StandardClaims
```

- **Description**: Parses an HS256 refresh token and extracts its claims.
- **Parameters**:
    - **`refreshToken`** (string): The refresh token to be parsed.
- **Returns**:
    - **`jwt.StandardClaims`**: The parsed standard claims.

### **GenerateOTP**

```go
func GenerateOTP(length int) string
```

- **Description**: Generates a random One-Time Password (OTP) of the specified length.
- **Parameters**:
    - **`length`** (int): The length of the OTP to be generated.
- **Returns**:
    - **`string`**: The randomly generated OTP.

### **GenerateRSAKeyPair**

```go
func GenerateRSAKeyPair() (PublicPrivatePair, error)
```

- **Description**: Generates a pair of RSA public and private keys with a key size of 2048 bits.
- **Returns**:
    - **`PublicPrivatePair`**: A struct containing the RSA public and private keys encoded in PEM format.
    - **`error`**: An error encountered during the key pair generation process, if any.

### **ValidateRSAKeyPair**

```go
func ValidateRSAKeyPair(keyPair PublicPrivatePair) (bool, error)
```

- **Description**: Validates an RSA key pair by encrypting and decrypting a message.
- **Parameters**:
    - **`keyPair`** (PublicPrivatePair): The RSA public and private keys encoded in PEM format.
- **Returns**:
    - **`bool`**: A boolean indicating whether the key pair is valid.
    - **`error`**: An error encountered during the validation process, if any.

### **HashSHA256**

```go
func HashSHA256(input string) string
```

- **Description**: Hashes the input string using SHA-256 algorithm.
- **Parameters**:
    - **`input`** (string): The input string to be hashed.
- **Returns**:
    - **`string`**: The hexadecimal representation of the hashed value.

### **VerifySHA256**

```go
func VerifySHA256(input, hashedValue string) bool
```

- **Description**: Verifies if the input matches the hashed value.
- **Parameters**:
    - **`input`** (string): The input string to be verified.
    - **`hashedValue`** (string): The hashed value to be compared with the hashed input.
- **Returns**:
    - **`bool`**: Returns true if the input matches the hashed value; otherwise, returns false.

### **GenerateSecureToken**

```go
func GenerateSecureToken(length int) (string, error)
```

- **Description**: Generates a secure token of the specified length.
- **Parameters**:
    - **`length`** (int): The length of the secure token to be generated.
- **Returns**:
    - **`string`**: The randomly generated secure token in hexadecimal format.
    - **`error`**: An error encountered during the token generation process, if any.

### **RetryGenerateSecureToken**

```go
func RetryGenerateSecureToken(length int, retries int) (string, error)
```

- **Description**: Retries generating a secure token with the specified length in case of failure.
- **Parameters**:
    - **`length`** (int): The length of the secure token to be generated.
    - **`retries`** (int): The number of retries allowed.
- **Returns**:
    - **`string`**: The randomly generated secure token in hexadecimal format.
    - **`error`**: An error encountered during the token generation process, if all retries fail.

### **GenerateRandomString**

```go
func GenerateRandomString(length int) string
```

- **Description**: Generates a random string of the specified length.
- **Parameters**:
    - **`length`** (int): The length of the random string to be generated.
- **Returns**:
    - **`string`**: The randomly generated string.

### **GenerateToken64**

```go
func GenerateToken64() string
```

- **Description**: Generates a base64 encoded token.
- **Returns**:
    - **`string`**: The base64 encoded token.

### **Types**

### **PublicPrivatePair**

```go
type PublicPrivatePair struct {
	PrivateKey string
	PublicKey  string
}
```

- **Description**: Contains the RSA public and private keys encoded in PEM format.

### **UserClaims**

```go
type UserClaims struct {
	First string `json:"first"`
	Last  string `json:"last"`
	Token string `json:"token"`
	jwt.StandardClaims
}
```

- **Description**: Represents user claims used in token generation.

### **Dependencies**

- **`crypto/rand`**
- **`encoding/base64`**
- **`encoding/hex`**
- **`encoding/json`**
- **`encoding/pem`**
- **`github.com/golang-jwt/jwt`**
- **`github.com/go-sql-driver/mysql`**
- **`github.com/google/uuid`**
- **`golang.org/x/crypto/argon2`**
- **`golang.org/x/crypto/bcrypt`**
- **`math/rand`**
- **`os`**
- **`strings`**
- **`time`**

### **Note**

- Ensure proper handling of errors returned by the functions to maintain the security and reliability of cryptographic operations.