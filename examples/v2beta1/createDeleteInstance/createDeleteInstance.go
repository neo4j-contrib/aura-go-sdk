// Package main demonstrates creating and deleting an Aura instance using the v2beta1 API.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/neo4j-contrib/aura-go-sdk/v2beta1"
)

func pollInstance(ctx context.Context, client *v2beta1.Client, instanceID string, want v2beta1.InstanceStatus) error {
	ctxPolling, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	i := 1
	for {
		select {
		case <-ctxPolling.Done():
			return errors.New("timed out waiting for instance status")
		default:
			result, err := client.Instances.Get(ctxPolling, instanceID)
			if err != nil {
				return err
			}
			if result.Data.Status == want {
				return nil
			}
			fmt.Printf("Instance status is %s. Poll no. %d\n", result.Data.Status, i)
			time.Sleep(1 * time.Second)
			i++
		}
	}
}

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
		v2beta1.WithOrganization(orgID),
		v2beta1.WithDefaultProject(projectID),
		v2beta1.WithTimeout(120*time.Second),
		v2beta1.WithLogger(customLogger),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()

	instances, err := client.Instances.List(ctx)
	if err != nil {
		log.Fatalf("Failed to list instances: %v", err)
	}

	for _, inst := range instances.Data {
		details, err := client.Instances.Get(ctx, inst.ID)
		if err != nil {
			log.Fatalf("Failed to get instance details for %s: %v", inst.ID, err)
		}
		if details.Data.Type == "free-db" {
			log.Fatalf("Instance %s (%s) is already using the free tier — cannot proceed", details.Data.Name, details.Data.ID)
		}
	}

	newInstance, err := client.Instances.Create(ctx, &v2beta1.CreateInstanceRequest{
		Name:          "auraClientV2beta1Example",
		Type:          "free-db",
		CloudProvider: "gcp",
		Region:        "europe-west1",
		Memory:        "1GB",
	})
	if err != nil {
		log.Fatalf("Failed to create instance: %v", err)
	}

	fmt.Println("\nInstance created")
	fmt.Printf("- %s: %s %s %s (%s)\n",
		newInstance.Data.Name,
		newInstance.Data.ID,
		newInstance.Data.CloudProvider,
		newInstance.Data.ConnectionURL,
		newInstance.Data.Username,
	)

	if err := pollInstance(ctx, client, newInstance.Data.ID, v2beta1.InstanceStatusRunning); err != nil {
		log.Fatalf("Instance did not reach running state: %v", err)
	}

	fmt.Printf("\nInstance %s is now running\n", newInstance.Data.Name)
	fmt.Println("Will delete in two seconds...")
	time.Sleep(2 * time.Second)

	_, err = client.Instances.Delete(ctx, newInstance.Data.ID)
	if err != nil {
		log.Fatalf("Failed to delete instance: %v", err)
	}

	fmt.Printf("\nInstance %s (%s) deleted\n", newInstance.Data.Name, newInstance.Data.ID)
	fmt.Println("\nv2beta1 client is working correctly!")
}
