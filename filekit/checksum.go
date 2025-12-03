package filekit

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"hash/crc32"
	"io"

	"github.com/cespare/xxhash/v2"
)

// NewHasher creates a new hash.Hash for the given algorithm.
// Returns an error if the algorithm is not supported.
func NewHasher(algorithm ChecksumAlgorithm) (hash.Hash, error) {
	switch algorithm {
	case ChecksumMD5:
		return md5.New(), nil
	case ChecksumSHA1:
		return sha1.New(), nil
	case ChecksumSHA256:
		return sha256.New(), nil
	case ChecksumSHA512:
		return sha512.New(), nil
	case ChecksumCRC32:
		return crc32.NewIEEE(), nil
	case ChecksumXXHash:
		return xxhash.New(), nil
	default:
		return nil, fmt.Errorf("%w: unsupported checksum algorithm: %s", ErrNotSupported, algorithm)
	}
}

// CalculateChecksum reads from the reader and calculates the checksum using
// the specified algorithm. Returns the hex-encoded checksum string.
func CalculateChecksum(r io.Reader, algorithm ChecksumAlgorithm) (string, error) {
	h, err := NewHasher(algorithm)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// CalculateChecksums reads from the reader and calculates multiple checksums
// in a single pass. Returns a map of algorithm to hex-encoded checksum.
func CalculateChecksums(r io.Reader, algorithms []ChecksumAlgorithm) (map[ChecksumAlgorithm]string, error) {
	if len(algorithms) == 0 {
		return nil, fmt.Errorf("no algorithms specified")
	}

	// Create hashers for each algorithm
	hashers := make(map[ChecksumAlgorithm]hash.Hash, len(algorithms))
	writers := make([]io.Writer, 0, len(algorithms))

	for _, algo := range algorithms {
		h, err := NewHasher(algo)
		if err != nil {
			return nil, err
		}
		hashers[algo] = h
		writers = append(writers, h)
	}

	// Create a multi-writer to write to all hashers at once
	multiWriter := io.MultiWriter(writers...)

	// Read the content once, writing to all hashers
	if _, err := io.Copy(multiWriter, r); err != nil {
		return nil, fmt.Errorf("failed to calculate checksums: %w", err)
	}

	// Collect results
	results := make(map[ChecksumAlgorithm]string, len(algorithms))
	for algo, h := range hashers {
		results[algo] = hex.EncodeToString(h.Sum(nil))
	}

	return results, nil
}

// VerifyChecksum downloads a file and verifies its checksum matches the expected value.
// This is a convenience function for integrity verification.
func VerifyChecksum(ctx context.Context, fs FileSystem, path string, expected string, algorithm ChecksumAlgorithm) (bool, error) {
	checksummer, ok := fs.(Checksummer)
	if !ok {
		return false, fmt.Errorf("%w: filesystem does not support checksums", ErrNotSupported)
	}

	actual, err := checksummer.Checksum(ctx, path, algorithm)
	if err != nil {
		return false, err
	}

	return actual == expected, nil
}
