// Package main demonstrates listing and creating organization invites using the v2beta1 API.
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
	inviteEmail := os.Getenv("AURA_INVITE_EMAIL")

	if clientID == "" || clientSecret == "" || orgID == "" || inviteEmail == "" {
		log.Fatal("Missing required environment variables: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_ORG_ID, AURA_INVITE_EMAIL")
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

	invites, err := client.OrganizationInvites.List(ctx, orgID)
	if err != nil {
		log.Fatalf("Failed to list organization invites: %v", err)
	}

	fmt.Printf("There are %d pending invites in organization %s\n\n", len(invites.Data), orgID)

	for _, inv := range invites.Data {
		fmt.Printf("- %s (status: %s, expires: %s)\n", inv.Email, inv.Status, inv.ExpiresAt)
	}

	fmt.Println()

	invite, err := client.OrganizationInvites.Create(ctx, orgID, &v2beta1.CreateOrganizationInviteRequest{
		Email: inviteEmail,
		Roles: []string{"organization:member"},
	})
	if err != nil {
		log.Fatalf("Failed to create organization invite: %v", err)
	}

	fmt.Printf("Created invite for %s (ID: %s, status: %s)\n", invite.Data.Email, invite.Data.ID, invite.Data.Status)

	fmt.Println("\nv2beta1 client is working correctly!")
}
