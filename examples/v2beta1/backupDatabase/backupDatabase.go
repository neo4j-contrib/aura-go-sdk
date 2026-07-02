// Package main demonstrates creating and deleting an Aura instance using the v2beta1 API.
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
	instanceID := os.Getenv("AURA_INSTANCE_ID")
	databaseID := os.Getenv("AURA_DATABASE_ID")

	if clientID == "" || clientSecret == "" || orgID == "" || projectID == "" || instanceID == "" {
		log.Fatal("Missing required environment variables: AURA_CLIENT_ID, AURA_CLIENT_SECRET, AURA_ORG_ID, AURA_PROJECT_ID, AURA_INSTANCE_ID, AURA_DATABASE_ID")
	}

	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	customLogger := slog.New(slog.NewTextHandler(os.Stderr, opts))

	client, err := v2beta1.NewClient(
		v2beta1.WithCredentials(clientID, clientSecret),
		v2beta1.WithDefaultOrg(orgID),
		v2beta1.WithDefaultProject(projectID),
		v2beta1.WithTimeout(120*time.Second),
		v2beta1.WithLogger(customLogger),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancelMain := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancelMain()

	fmt.Printf("\\nBacking up database")

	newBackup, err := client.DatabaseBackups.Create(ctx, instanceID, databaseID)
	if err != nil {
		log.Fatalf("Failed to backup database: %v", err)
	}
	fmt.Println("\nDatabase backup started")
	fmt.Printf("- backup id: %s \n", newBackup.Data.ID)

	getBackup, err := client.DatabaseBackups.Get(ctx, instanceID, databaseID, newBackup.Data.ID)

	fmt.Printf("backup information: %s \n", getBackup.Data.ID)

	fmt.Println("\nv2beta1 client is working correctly!")
}
