// Package main demonstrates listing and adding project users using the v2beta1 API.
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
	projectID := os.Getenv("AURA_PROJECT_ID")

	if clientID == "" || clientSecret == "" || orgID == "" || projectID == "" {
		log.Fatal("Missing required environment variables: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_ORG_ID, AURA_PROJECT_ID")
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users, err := client.ProjectUsers.List(ctx, orgID, projectID)
	if err != nil {
		log.Fatalf("Failed to list project users: %v", err)
	}

	fmt.Printf("Project %s has %d user(s):\n", projectID, len(users.Data))
	for _, u := range users.Data {
		fmt.Printf("- %s (%s): %v\n", u.Email, u.UserID, u.ProjectRoles)
	}

	userIDToAdd := os.Getenv("AURA_USER_ID")
	if userIDToAdd == "" {
		fmt.Println("\nSet AURA_USER_ID to demonstrate adding a user to the project.")
		fmt.Println("\nv2beta1 client is working correctly!")
		return
	}

	err = client.ProjectUsers.Add(ctx, orgID, projectID, &v2beta1.AddProjectUserRequest{
		UserID:       userIDToAdd,
		ProjectRoles: []string{"viewer"},
	})
	if err != nil {
		log.Fatalf("Failed to add user to project: %v", err)
	}

	fmt.Printf("\nUser %s added to project %s\n", userIDToAdd, projectID)
	fmt.Println("\nv2beta1 client is working correctly!")
}
