// Package main demonstrates listing and retrieving organization users using the v2beta1 API.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/v2beta1"
)

func main() {
	clientID := os.Getenv("AURA_CLIENT_ID")
	clientSecret := os.Getenv("AURA_CLIENT_SECRET")
	orgID := os.Getenv("AURA_ORG_ID")
	userID := os.Getenv("AURA_USER_ID")

	if clientID == "" || clientSecret == "" || orgID == "" || userID == "" {
		log.Fatal("Missing required environment variables: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_ORG_ID, AURA_USER_ID")
	}

	opts := &slog.HandlerOptions{Level: slog.LevelWarn}
	customLogger := slog.New(slog.NewTextHandler(os.Stderr, opts))

	client, err := v2beta1.NewClient(
		v2beta1.WithCredentials(clientID, clientSecret),
		v2beta1.WithTimeout(120*time.Second),
		v2beta1.WithLogger(customLogger),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	users, err := client.OrganizationUsers.List(ctx, orgID)
	if err != nil {
		log.Fatalf("Failed to list organization users: %v", err)
	}

	fmt.Printf("There are %d users in organization %s\n\n", len(users.Data), orgID)

	for _, u := range users.Data {
		fmt.Printf("- %s (%s): roles=%v\n", u.Email, u.UserID, u.OrganizationRoles)
	}

	fmt.Println()

	user, err := client.OrganizationUsers.Get(ctx, orgID, userID)
	if err != nil {
		log.Fatalf("Failed to get organization user: %v", err)
	}

	fmt.Printf("User details for %s:\n", user.Data.Email)
	for _, project := range user.Data.Projects {
		fmt.Printf("  - Project %s (%s): roles=%v\n", project.Name, project.ID, project.ProjectRoles)
	}

	fmt.Println("\nv2beta1 client is working correctly!")
}
