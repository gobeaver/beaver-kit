package filekit

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"os"
)

// EncryptedFS is a wrapper around a FileSystem that encrypts and decrypts data
type EncryptedFS struct {
	fs  FileSystem
	key []byte
}

// NewEncryptedFS creates a new encrypted filesystem
func NewEncryptedFS(fs FileSystem, key []byte) *EncryptedFS {
	// Ensure key is 32 bytes (for AES-256)
	if len(key) != 32 {
		panic("encryption key must be 32 bytes")
	}

	return &EncryptedFS{
		fs:  fs,
		key: key,
	}
}

// Upload encrypts the content before uploading
func (e *EncryptedFS) Upload(ctx context.Context, path string, content io.Reader, options ...Option) error {
	// Create a pipe for streaming
	pr, pw := io.Pipe()

	// Start encryption in a separate goroutine
	go func() {
		var err error
		defer func() {
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		}()

		// Create a new AES cipher
		block, err := aes.NewCipher(e.key)
		if err != nil {
			return
		}

		// Create a new GCM cipher
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return
		}

		// Create a nonce
		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
			return
		}

		// Write the nonce to the pipe
		if _, err = pw.Write(nonce); err != nil {
			return
		}

		// Create a buffer for reading from the input
		buf := make([]byte, 32*1024)

		// Read and encrypt in chunks
		for {
			n, err := content.Read(buf)
			if err != nil && err != io.EOF {
				return
			}

			if n > 0 {
				// Encrypt the data
				ciphertext := gcm.Seal(nil, nonce, buf[:n], nil)

				// Write the encrypted data to the pipe
				if _, err := pw.Write(ciphertext); err != nil {
					return
				}
			}

			if err == io.EOF {
				break
			}
		}
	}()

	// Upload the encrypted data
	return e.fs.Upload(ctx, path, pr, options...)
}

// Download decrypts the content after downloading
func (e *EncryptedFS) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	// Download the encrypted content
	encryptedContent, err := e.fs.Download(ctx, path)
	if err != nil {
		return nil, err
	}

	// Create a new AES cipher
	block, err := aes.NewCipher(e.key)
	if err != nil {
		encryptedContent.Close()
		return nil, err
	}

	// Create a new GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		encryptedContent.Close()
		return nil, err
	}

	// Create a pipe for streaming decrypted data
	pr, pw := io.Pipe()

	// Start decryption in a separate goroutine
	go func() {
		var err error
		defer func() {
			encryptedContent.Close()
			if err != nil {
				pw.CloseWithError(err)
			} else {
				pw.Close()
			}
		}()

		// Read the nonce
		nonce := make([]byte, gcm.NonceSize())
		if _, err = io.ReadFull(encryptedContent, nonce); err != nil {
			return
		}

		// Create a buffer for decryption
		buf := make([]byte, 32*1024+gcm.Overhead())

		// Keep track of any leftover bytes from previous read
		var leftover []byte

		// Read and decrypt in chunks
		for {
			n, err := encryptedContent.Read(buf)
			if err != nil && err != io.EOF {
				return
			}

			if n > 0 {
				// Combine leftover with current read
				data := append(leftover, buf[:n]...)

				// We need at least one block to decrypt
				if len(data) < gcm.Overhead() {
					leftover = data
					if err == io.EOF {
						// If we're at EOF and don't have enough data, something is wrong
						err = errors.New("invalid encrypted data")
						return
					}
					continue
				}

				// Try to decrypt as much as we can
				plaintext, err := gcm.Open(nil, nonce, data, nil)
				if err != nil {
					return
				}

				// Write the decrypted data to the pipe
				if _, err := pw.Write(plaintext); err != nil {
					return
				}

				leftover = nil
			}

			if err == io.EOF {
				break
			}
		}
	}()

	return pr, nil
}

// Delete delegates to the underlying filesystem
func (e *EncryptedFS) Delete(ctx context.Context, path string) error {
	return e.fs.Delete(ctx, path)
}

// Exists delegates to the underlying filesystem
func (e *EncryptedFS) Exists(ctx context.Context, path string) (bool, error) {
	return e.fs.Exists(ctx, path)
}

// FileInfo delegates to the underlying filesystem
func (e *EncryptedFS) FileInfo(ctx context.Context, path string) (*File, error) {
	return e.fs.FileInfo(ctx, path)
}

// List delegates to the underlying filesystem
func (e *EncryptedFS) List(ctx context.Context, prefix string) ([]File, error) {
	return e.fs.List(ctx, prefix)
}

// CreateDir delegates to the underlying filesystem
func (e *EncryptedFS) CreateDir(ctx context.Context, path string) error {
	return e.fs.CreateDir(ctx, path)
}

// DeleteDir delegates to the underlying filesystem
func (e *EncryptedFS) DeleteDir(ctx context.Context, path string) error {
	return e.fs.DeleteDir(ctx, path)
}

// UploadFile encrypts and uploads a local file
func (e *EncryptedFS) UploadFile(ctx context.Context, path, localPath string, options ...Option) error {
	// Open the local file
	file, err := os.Open(localPath)
	if err != nil {
		return &PathError{
			Op:   "uploadfile",
			Path: localPath,
			Err:  err,
		}
	}
	defer file.Close()

	// Upload the file
	return e.Upload(ctx, path, file, options...)
}
