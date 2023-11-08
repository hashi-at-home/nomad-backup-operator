package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/nomad/api"
)

// create NodeConsumer struct containing client and onNode function
type NodeConsumer struct {
	client *api.Client
	onNode func(eventType string, node *api.Node)
	stop   func()
}

// NewNodeConsumer is a function which consumes the client pointer and onNode function and returns a consumer address
func NewNodeConsumer(client *api.Client, onNode func(eventType string, node *api.Node)) *NodeConsumer {
	return &NodeConsumer{
		client: client,
		onNode: onNode,
	}
}

// nStop is a function of type pointer to Consumer which stops the consumer
func (n *NodeConsumer) nStop() {
	if n.stop != nil {
		n.stop()
	}
}

func (n *NodeConsumer) Start() {
	ctx := context.Background()
	ctx, n.stop = context.WithCancel(ctx)

	n.consumeNode(ctx)
}

// consume is a function of type pointer to Consumer which takes as input a context and returns and error
func (c *NodeConsumer) consumeNode(ctx context.Context) error {
	// index is a variable of type unit64 which tracks the job's index
	var index uint64 = 0
	// get the job list and set index to the last job's index
	if _, meta, err := c.client.Nodes().List(nil); err == nil {
		index = meta.LastIndex
	}

	// get all the job event topics
	topics := map[api.Topic][]string{
		api.TopicNode: {"*"},
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
			c.handleNodeEvent(event)
		}
	}
}

// handleEvent is a function of type pointer to Consumer
// which takes a pointer to a list of API events as input
// and returns a void
func (c *NodeConsumer) handleNodeEvent(event *api.Events) {
	if event.Err != nil {
		fmt.Printf("received error %s\n", event.Err)
		return
	}

	// loop over events
	for _, e := range event.Events {

		// Get the job from the event
		node, err := e.Node()
		if err != nil {
			fmt.Printf("received error %s\n", err)
			return
		}

		// ignore nil jobs
		if node == nil {
			return
		}

		// log the event
		fmt.Printf("==> %s: %s (%s)...\n", e.Type, node.Name, node.ID)

		// call the onJob function
		c.onNode(e.Type, node)
	}
}
