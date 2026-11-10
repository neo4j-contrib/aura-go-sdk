// Package main demonstrates listing organizations using the v2beta1 API.
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

	if clientID == "" || clientSecret == "" {
		log.Fatal("Missing required environment variables: AURA_CLIENT_ID, AURA_CLIENT_SECRET ")
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

	organizations, err := client.Organizations.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list organizations: %v", err)
	}

	orgCount := len(organizations.Data)

	fmt.Printf("There are %v organizations in total \n\n", orgCount)

	for _, org := range organizations.Data {
		fmt.Printf("- %s: %s\n",
			org.Name,
			org.ID)

		// Get projects for the organisation
		projects, err := client.Projects.List(ctx, v2beta1.WithOrg(org.ID))
		if err != nil {
			log.Fatalf("Failed to list projects for organization: %v", err)
		}

		fmt.Printf("-- Projects \n")

		for _, project := range projects.Data {
			fmt.Printf("-- %s, %s\n",
				project.Name,
				project.ID)
		}

		fmt.Printf("\n\n")
	}

	fmt.Println("\nv2beta1 client is working correctly!")
}
