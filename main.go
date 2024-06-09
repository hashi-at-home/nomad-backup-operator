package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/nomad/api"
	vault "github.com/hashicorp/vault/api"
)

func main() {
	log.SetOutput(os.Stdout)
	if err := run(os.Args[:]); err != nil {
		fmt.Println(err)
		log.Fatal(err)
		os.Exit(1)
	}
}

func run(args []string) error {

	// create Vault client

	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = os.Getenv("VAULT_ADDR")

	vaultClient, err := vault.NewClient(vaultConfig)

	if err != nil {
		log.Fatal("Unable to initialize Vault")
	}

	vaultClient.SetToken(os.Getenv("VAULT_TOKEN"))

	// create Nomad client by calling NewClient()
	client, err := api.NewClient(&api.Config{Address: "http://bare:4646", Namespace: "default"})
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
	consumer := NewConsumer(client, backup.OnJob)
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

	// start the consumer
	consumer.Start()
	return nil
}
