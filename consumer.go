// Nomad Streaming API consumer

package main

import (
	"context"
	"fmt"

	nomad "github.com/hashicorp/nomad/api"
	vault "github.com/hashicorp/vault/api"
)

// create Consumer struct containing client and onJob function
type Consumer struct {
	client *nomad.Client
	onJob  func(eventType string, job *nomad.Job)
	vault  *vault.Client
	stop   func()
}

type NomadCredentials struct {
	AccessorId string `json:"accessor_id"`
	SecretId   string `json:"secret_id"`
}

// NewConsumer is a function which consumes the client pointer and onJob function and returns a Consumer address
func NewConsumer(client *nomad.Client, onJob func(eventType string, job *nomad.Job), vault *vault.Client) *Consumer {
	return &Consumer{
		client: client,
		onJob:  onJob,
		vault:  vault,
	}
}

// Stop is a function of type pointer to Consumer which stops the consumer
func (c *Consumer) Stop() {
	if c.stop != nil {
		c.stop()
	}
}

// Start is a function of type pointer to consumer and starts a context in the background
func (c *Consumer) Start(vault *vault.Client) {
	ctx := context.Background()
	ctx, c.stop = context.WithCancel(ctx)
	fmt.Println("Starting")
	c.consume(ctx, vault)
}

// consume is a function of type pointer to Consumer which takes as input a context and returns and error
func (c *Consumer) consume(ctx context.Context, vault *vault.Client) error {
	secret, err := vault.Logical().ReadWithContext(ctx, "nomad/creds/backup-operator")
	if err != nil {
		return fmt.Errorf("Error getting nomad token %w", err)
	}

	if secret != nil { // pragma: allowlist secret
		return fmt.Errorf("Nomad Credentials not returned from Vault")
	}

	secretID, ok := secret.Data["secret_id"].(string)
	if !ok {
		return fmt.Errorf("Nomad Secret ID not returned from Vault response")
	}
	c.client.SetSecretID(secretID)

	// index is a variable of type unit64 which tracks the job's index
	var index uint64 = 0
	// get the job list and set index to the last job's index
	q := nomad.QueryOptions{
		Namespace: "default",
		Region:    "global",
		Headers: map[string]string{
			"X-Nomad-Token": secretID},
	}
	if _, meta, err := c.client.Jobs().List(&q); err == nil {
		index = meta.LastIndex
	}

	fmt.Printf("Index is now %d", index)
	// get all the job event topics
	topics := map[nomad.Topic][]string{
		nomad.TopicJob: {"*"},
	}
	// create new Nomad events client
	eventsClient := c.client.EventStream()
	// create new stream channel
	eventCh, err := eventsClient.Stream(ctx, topics, index, &q)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil

		case event := <-eventCh:
			// ignore heartbeats used to keep connection alive
			if event.IsHeartbeat() {
				continue
			}
			fmt.Println("Handling event")
			c.handleEvent(event)
		}
	}
}

// handleEvent is a function of type pointer to Consumer
// which takes a pointer to a list of API events as input
// and returns a void
func (c *Consumer) handleEvent(event *nomad.Events) {
	if event.Err != nil {
		fmt.Printf("received error %s\n", event.Err)
		return
	}

	// loop over events
	for _, e := range event.Events {
		// ignore event types that are not of type JobRegistered or JobDeregistered
		if e.Type != "JobRegistered" && e.Type != "JobDeregistered" {
			return
		}

		// Get the job from the event
		job, err := e.Job()
		if err != nil {
			fmt.Printf("received error %s\n", err)
			return
		}

		// ignore nil jobs
		if job == nil {
			return
		}

		// log the event
		fmt.Printf("==> %s: %s (%s)...\n", e.Type, *job.ID, *job.Status)

		// call the onJob function
		c.onJob(e.Type, job)
	}
}
