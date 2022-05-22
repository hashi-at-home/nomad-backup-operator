package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/nomad/api"
)

func main() {
	if err := run(os.Args[:]); err != nil {
		fmt.Println(err)
		log.Fatal(err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// create Nomad client by calling NewClient()
	client, err := api.NewClient(&api.Config{})
	fmt.Println("Starting Nomad backup operator")
	if err != nil {
		return err
	}

	backup := NewBackup(client)
	consumer := NewConsumer(client, backup.OnJob)

	// create signals channel to listen for signals sent to us from the OS
	signals := make(chan os.Signal, 1)
	// Watch for SIGINT or SIGTERM signals
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

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
