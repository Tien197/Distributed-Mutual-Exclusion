package main

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"sync"
	"time"

	pb "Module/Proto"
)

// Node represents a node in the distributed system
type Node struct {
	ID                    int64
	Clock                 int64
	Acknowledgements      int
	RequestingCS          bool
	DeferredReplies       []int64
	Mutex                 sync.Mutex
	OtherNodes            []string // List of other node addresses
	MutualExclusionClient pb.MutualExclusionServiceClient
}

// Request represents a request message
type Request struct {
	NodeID int64
	Clock  int64
}

// NewNode initializes a new Node
func NewNode(id int64, otherNodes []string) *Node {
	return &Node{
		ID:              id,
		Clock:           0,
		OtherNodes:      otherNodes,
		RequestingCS:    false,
		DeferredReplies: make([]int64, 0),
	}
}

// SendRequest is called when the node wants to enter the critical section
func (n *Node) SendRequest() {
	n.Mutex.Lock()
	n.Clock++
	n.RequestingCS = true
	n.Acknowledgements = 0
	n.Mutex.Unlock()

	req := &pb.Request{
		NodeId: n.ID,
		Clock:  n.Clock,
	}

	// Send request to all other nodes
	for _, addr := range n.OtherNodes {
		conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect: %v", err)
		}
		defer conn.Close()
		c := pb.NewMutualExclusionServiceClient(conn)

		// Asynchronous call
		go func() {
			_, err := c.ReceiveRequest(context.Background(), req)
			if err != nil {
				log.Fatalf("could not greet: %v", err)
			}
		}()
	}

	// Wait for acknowledgements from all other nodes
	for {
		n.Mutex.Lock()
		if n.Acknowledgements == len(n.OtherNodes) {
			break
		}
		n.Mutex.Unlock()
		time.Sleep(1 * time.Second) // Wait a bit before checking again
	}

	// Now enter critical section
	n.Mutex.Unlock()
	n.EnterCriticalSection()
	n.LeaveCriticalSection()
}

// EnterCriticalSection is where the critical section logic is implemented
func (n *Node) EnterCriticalSection() {
	log.Println("Node", n.ID, "entering critical section")
	fmt.Println("I am in the critical section", n.ID)
	time.Sleep(2 * time.Second) // Simulate some operation
	log.Println("Node", n.ID, "leaving critical section")
}

// LeaveCriticalSection is called when leaving the critical section
func (n *Node) LeaveCriticalSection() {
	n.Mutex.Lock()
	n.RequestingCS = false
	n.Mutex.Unlock()

	// Send reply to all deferred requests
	for _, nodeID := range n.DeferredReplies {
		//TODO Logic to send reply to the deferred requests
	}
	n.DeferredReplies = []int64{}
}

// ReceiveRequest handles incoming requests from other nodes
func (n *Node) ReceiveRequest(ctx context.Context, req *pb.Request) (*pb.Reply, error) {
	n.Mutex.Lock()
	defer n.Mutex.Unlock()

	// Update clock
	n.Clock = max(n.Clock, req.Clock) + 1

	if n.RequestingCS && (req.Clock > n.Clock || (req.Clock == n.Clock && req.NodeId > n.ID)) {
		// Defer reply
		n.DeferredReplies = append(n.DeferredReplies, req.NodeId)
	} else {
		// Send immediate reply
		//TODO Logic to send reply
	}

	return &pb.Reply{NodeId: n.ID}, nil
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
