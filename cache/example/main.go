package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"securedesign/cache"
)

type User struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	LastSeen time.Time `json:"last_seen"`
}

func main() {
	// Initialize the Redis cache
	redisCache, err := cache.NewRedisCache(cache.RedisConfig{
		Address:  "localhost:6379", // Change to your Redis server address
		Password: "",               // Set if Redis requires authentication
		DB:       0,                // Default Redis DB
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisCache.Close()

	// Create a context
	ctx := context.Background()

	// Create a user to cache
	user := User{
		ID:       "user123",
		Name:     "John Doe",
		Email:    "john@example.com",
		LastSeen: time.Now(),
	}

	// Cache the user for 1 hour
	err = redisCache.Set(ctx, "user:"+user.ID, user, time.Hour)
	if err != nil {
		log.Fatalf("Failed to cache user: %v", err)
	}
	fmt.Println("User cached successfully")

	// Retrieve the user from cache
	var cachedUser User
	err = redisCache.Get(ctx, "user:"+user.ID, &cachedUser)
	if err != nil {
		if err == cache.ErrKeyNotFound {
			log.Println("User not found in cache")
		} else {
			log.Fatalf("Failed to get user from cache: %v", err)
		}
	} else {
		fmt.Printf("Retrieved user from cache: %+v\n", cachedUser)
	}

	// Check if a key exists
	exists, err := redisCache.Exists(ctx, "user:"+user.ID)
	if err != nil {
		log.Fatalf("Failed to check if key exists: %v", err)
	}
	fmt.Printf("User exists in cache: %v\n", exists)

	// Delete the user from cache
	err = redisCache.Delete(ctx, "user:"+user.ID)
	if err != nil {
		log.Fatalf("Failed to delete user from cache: %v", err)
	}
	fmt.Println("User deleted from cache")

	// Verify the user is deleted
	exists, err = redisCache.Exists(ctx, "user:"+user.ID)
	if err != nil {
		log.Fatalf("Failed to check if key exists: %v", err)
	}
	fmt.Printf("User exists in cache after deletion: %v\n", exists)
}
