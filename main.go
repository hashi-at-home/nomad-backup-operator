package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	nomad "github.com/hashicorp/nomad/api"
	vault "github.com/hashicorp/vault/api"

	"github.com/charmbracelet/log"
)

func main() {
	log.SetOutput(os.Stdout)

	if err := run(context.Background()); err != nil {
		fmt.Println(err)
		log.Fatal(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {

	// configure Nomad Address
	nomadAddr := os.Getenv("NOMAD_ADDR")

	if nomadAddr == "" {
		// set nomad addr to local agent if it's not set.
		nomadAddr = "http://localhost:4647"
	}
	snapshotOutDir := os.Getenv("SNAPSHOT_DIR")
	if snapshotOutDir == "" {
		snapshotOutDir = "."
	}

	nomadToken := os.Getenv("NOMAD_TOKEN")
	if nomadToken == "" {
		return fmt.Errorf("NOMAD_TOKEN not set.")
	}
	// create Vault client
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = os.Getenv("VAULT_ADDR")

	vaultClient, err := vault.NewClient(vaultConfig)
	if err != nil {
		log.Fatal("Unable to initialize Vault")
	}

	vaultClient.SetToken(os.Getenv("VAULT_TOKEN"))

	// create Nomad client by calling NewClient()
	client, err := nomad.NewClient(&nomad.Config{Address: os.Getenv("NOMAD_ADDR"), Namespace: "default"})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Starting Nomad backup operator")
	peers, err := client.Status().Peers()
	fmt.Printf("There are %d Peers\n", len(peers))
	for k, p := range peers {
		fmt.Printf("Peer: %d is %s\n", k, p)
	}

	if err != nil {
		fmt.Println(err)
		return err
	}

	backup := NewBackup(client, vaultClient)
	consumer := NewConsumer(client, backup.OnJob, vaultClient)
	fmt.Println(backup.client.Address())

	// create signals channel to listen for signals sent to us from the OS
	fmt.Println("Making channel")
	signals := make(chan os.Signal, 1)
	// Watch for SIGINT or SIGTERM signals
	fmt.Println(len(signals))
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// anonymous shutdown function.
	go func() {
		s := <-signals
		fmt.Printf("Received %s, stopping\n", s)
		consumer.Stop()
		os.Exit(0)
	}()

	// start the consumer with the vault client
	consumer.Start(vaultClient)
	return nil
}
