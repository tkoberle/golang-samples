package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// MockRegistry holds the request hash to response file mapping
type MockRegistry struct {
	Mapping       map[string]string
	ResponseDir   string
	ResponseTypes map[string]proto.Message
	cache         sync.Map // Cache for previously seen requests
}

// LoadRegistry loads the mapping file and initializes the registry
func LoadRegistry(mappingFilePath, responseDir string, responseTypes map[string]proto.Message) (*MockRegistry, error) {
	data, err := os.ReadFile(mappingFilePath)
	if err != nil {
		return nil, err
	}
	var mapping map[string]string
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}
	return &MockRegistry{Mapping: mapping, ResponseDir: responseDir, ResponseTypes: responseTypes}, nil
}

// GetResponse retrieves a mocked response for a given request
func (r *MockRegistry) GetResponse(req proto.Message) (proto.Message, error) {
	jsonBytes, err := protojson.MarshalOptions{
		EmitUnpopulated: true,
	}.Marshal(req)
	if err != nil {
		return nil, err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(jsonBytes))
	if val, ok := r.cache.Load(hash); ok {
		return val.(proto.Message), nil
	}

	responseFile, ok := r.Mapping[hash]
	if !ok {
		return nil, fmt.Errorf("no mock response found for hash: %s", hash)
	}

	responsePath := filepath.Join(r.ResponseDir, responseFile)
	respData, err := os.ReadFile(responsePath)
	if err != nil {
		return nil, err
	}

	responseTypeName := inferTypeName(responseFile)
	typeTemplate, ok := r.ResponseTypes[responseTypeName]
	if !ok {
		return nil, fmt.Errorf("unknown response type: %s", responseTypeName)
	}

	response := proto.Clone(typeTemplate)
	if err := protojson.Unmarshal(respData, response); err != nil {
		return nil, err
	}

	r.cache.Store(hash, response)
	return response, nil
}

// inferTypeName infers the type name from the response file name (basic version)
func inferTypeName(filename string) string {
	base := filepath.Base(filename)
	return base[:len(base)-len(filepath.Ext(base))] // strip extension
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()

	// Example: mypb.RegisterMyServiceServer(grpcServer, &MyMockServer{})

	go func() {
		log.Println("Mock gRPC server listening on :50051")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gRPC server...")
	grpcServer.GracefulStop()
}

// Define your mock service implementation as needed. For example:
// type MyMockServer struct {
//     mypb.UnimplementedMyServiceServer
//     Registry *MockRegistry
// }
// func (s *MyMockServer) MyMethod(ctx context.Context, req *mypb.MyRequest) (*mypb.MyResponse, error) {
//     return s.Registry.GetResponse(req)
// }
