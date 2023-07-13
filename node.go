package main

import (
	"fmt"

	"github.com/hashicorp/nomad/api"
)

var AnsibleHCL string

// Node is a struct containing a pointer to a nomad API client
type Node struct {
	client *api.Client
}

// NewNode is a function which takes a pointer to a nomad API client
// and returns the address of the Node
func NewNode(client *api.Client) *Node {
	return &Node{
		client: client,
	}
}

// OnNode is a function of type pointer to Configure which takes a string and a pointer to a
// Nomad node and returns a nil
// It will configure Turing nodes
func (n *Node) onNode(eventType string, node *api.Node) {
	// if strings.HasPrefix(node.Name, "turing-")
	fmt.Println(node.Name)
}
