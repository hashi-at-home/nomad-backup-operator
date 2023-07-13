package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/nomad/api"
)

// create Consumer struct containing client and onJob function
type Consumer struct {
	client *api.Client
	onJob  func(eventType string, job *api.Job)
	stop   func()
}

// NewConsumer is a function which consumes the client pointer and onJob function and returns a Consumer address
func NewConsumer(client *api.Client, onJob func(eventType string, job *api.Job)) *Consumer {
	return &Consumer{
		client: client,
		onJob:  onJob,
	}
}

// Stop is a function of type pointer to Consumer which stops the consumer
func (c *Consumer) Stop() {
	if c.stop != nil {
		c.stop()
	}
}

// Start is a function of type pointer to consumer and starts a context in the background
func (c *Consumer) Start() {
	ctx := context.Background()
	ctx, c.stop = context.WithCancel(ctx)

	c.consume(ctx)
}

// consume is a function of type pointer to Consumer which takes as input a context and returns and error
func (c *Consumer) consume(ctx context.Context) error {
	// index is a variable of type unit64 which tracks the job's index
	var index uint64 = 0
	// get the job list and set index to the last job's index
	if _, meta, err := c.client.Jobs().List(nil); err == nil {
		index = meta.LastIndex
	}

	// get all the job event topics
	topics := map[api.Topic][]string{
		api.TopicJob: {"*"},
	}

	// create new Nomad events client
	eventsClient := c.client.EventStream()
	// create new stream channel
	eventCh, err := eventsClient.Stream(ctx, topics, index, &api.QueryOptions{})
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
			c.handleEvent(event)
		}
	}
}

// handleEvent is a function of type pointer to Consumer
// which takes a pointer to a list of API events as input
// and returns a void
func (c *Consumer) handleEvent(event *api.Events) {
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
