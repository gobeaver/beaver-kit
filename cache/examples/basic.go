package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gobeaver/beaver-kit/cache"
)

func main() {
	// Initialize cache from environment variables
	// Set BEAVER_CACHE_DRIVER=memory or BEAVER_CACHE_DRIVER=redis
	if err := cache.Init(); err != nil {
		log.Fatal(err)
	}
	defer cache.Shutdown(context.Background())

	ctx := context.Background()

	// Example: User session management
	sessionID := "user-123-session"
	sessionData := []byte(`{"user_id": "123", "email": "user@example.com"}`)

	// Store session with 30 minute TTL
	fmt.Println("Storing user session...")
	if err := cache.Set(ctx, sessionID, sessionData, 30*time.Minute); err != nil {
		log.Fatal(err)
	}

	// Retrieve session
	fmt.Println("Retrieving user session...")
	data, err := cache.Get(ctx, sessionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Session data: %s\n", string(data))

	// Check if session exists
	exists, err := cache.Exists(ctx, sessionID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Session exists: %v\n", exists)

	// Example: API rate limiting
	rateLimitKey := "api-rate-limit:user-123"
	requestCount := []byte("1")

	// Set rate limit counter with 1 minute window
	fmt.Println("\nSetting API rate limit counter...")
	if err := cache.Set(ctx, rateLimitKey, requestCount, 1*time.Minute); err != nil {
		log.Fatal(err)
	}

	// Check health
	if cache.IsHealthy() {
		fmt.Println("\nCache is healthy!")
	}

	// Clean up
	fmt.Println("\nCleaning up...")
	cache.Delete(ctx, sessionID)
	cache.Delete(ctx, rateLimitKey)

	fmt.Println("\nDone! The same code works with both memory and Redis drivers.")
	fmt.Println("Switch drivers by setting BEAVER_CACHE_DRIVER environment variable.")
}
