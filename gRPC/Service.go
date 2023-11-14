package gRPC

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	pb "Module/Proto" // Replace with your actual generated package path
	"google.golang.org/grpc"
)

// Node represents a node in the distributed system
type Node struct {
	pb.UnimplementedMutualExclusionServiceServer

	ID               int64
	Clock            int64
	Acknowledgements int
	RequestingCS     bool
	DeferredReplies  []int64
	Mutex            sync.Mutex
	OtherNodes       []string // List of other node addresses
	grpcClient       map[string]pb.MutualExclusionServiceClient
}

// NewNode initializes a new Node with gRPC clients
func NewNode(id int64, otherNodes []string) *Node {
	node := &Node{
		ID:              id,
		Clock:           0,
		OtherNodes:      otherNodes,
		RequestingCS:    false,
		DeferredReplies: make([]int64, 0),
		grpcClient:      make(map[string]pb.MutualExclusionServiceClient),
	}

	// Set up a connection to the gRPC server for each other node.
	for _, addr := range otherNodes {
		conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Fatalf("did not connect to %v: %v", addr, err)
		}
		node.grpcClient[addr] = pb.NewMutualExclusionServiceClient(conn)
	}

	return node
}

// SendRequest is updated to use gRPC for communication
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

	// Send request to all other nodes using gRPC
	for _, addr := range n.OtherNodes {
		go func(address string) {
			_, err := n.grpcClient[address].ReceiveRequest(context.Background(), req)
			if err != nil {
				log.Fatalf("Error sending request to %v: %v", address, err)
			}
			n.Mutex.Lock()
			n.Acknowledgements++
			n.Mutex.Unlock()
		}(addr)
	}

	// Wait for acknowledgements from all other nodes
	for {
		n.Mutex.Lock()
		if n.Acknowledgements == len(n.OtherNodes) {
			n.Mutex.Unlock()
			break
		}
		n.Mutex.Unlock()
		time.Sleep(1 * time.Second) // Wait a bit before checking again
	}

	// Now enter critical section
	n.EnterCriticalSection()
	n.LeaveCriticalSection()
}

// EnterCriticalSection is where the critical section logic is implemented
func (n *Node) EnterCriticalSection() {
	// Critical section code goes here
	log.Println("Entering Critical Section")
	// Simulate critical section work
	time.Sleep(2 * time.Second)
	log.Println("Leaving Critical Section")
}

// LeaveCriticalSection is called when leaving the critical section
func (n *Node) LeaveCriticalSection() {
	n.Mutex.Lock()
	n.RequestingCS = false

	// Send reply to all deferred requests
	for _, nodeID := range n.DeferredReplies {
		//TODO Logic to send reply to the deferred requests
	}
	n.DeferredReplies = []int64{}
	n.Mutex.Unlock()
}

// ReceiveRequest handles incoming requests from other nodes (gRPC server side)
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

// StartServer initializes and starts a gRPC server for the node
func (n *Node) StartServer(address string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMutualExclusionServiceServer(s, n)

	log.Printf("Server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	//TODO
	// Example Node initialization and server setup
	// node := NewNode(1, []string{"localhost:50051", "localhost:50052"})
	// go node.StartServer("localhost:50050")
	// ... other initialization and logic
}
