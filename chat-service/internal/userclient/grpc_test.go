package userclient

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/ride-sharing/shared/proto/user"
)

type mockUserServiceServer struct {
	user.UnimplementedUserServiceServer
	blockedUsers map[string]bool
}

func (s *mockUserServiceServer) CheckBlockStatus(ctx context.Context, req *user.BlocKStatusCheckRequest) (*user.BlocKStatusCheckResponse, error) {
	key := req.SenderId + ":" + req.RecId
	if _, ok := s.blockedUsers[key]; ok {
		return &user.BlocKStatusCheckResponse{IsBlocked: true}, nil
	}
	reverseKey := req.RecId + ":" + req.SenderId
	if _, ok := s.blockedUsers[reverseKey]; ok {
		return &user.BlocKStatusCheckResponse{IsBlocked: true}, nil
	}
	return &user.BlocKStatusCheckResponse{IsBlocked: false}, nil
}

func (s *mockUserServiceServer) GetInternalProfile(ctx context.Context, req *user.InternalProfileRequest) (*user.InternalProfileResponse, error) {
	return &user.InternalProfileResponse{
		Id: req.UserId,
	}, nil
}

func setupTestClient(blocked map[string]bool) (*Client, func(), error) {
	buffer := 1024 * 1024
	listener := bufconn.Listen(buffer)

	srv := grpc.NewServer()
	mockSrv := &mockUserServiceServer{blockedUsers: blocked}
	user.RegisterUserServiceServer(srv, mockSrv)

	go srv.Serve(listener)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		srv.Stop()
		return nil, nil, err
	}

	client := &Client{
		conn: conn,
		stub: user.NewUserServiceClient(conn),
	}

	cleanup := func() {
		conn.Close()
		srv.Stop()
	}

	return client, cleanup, nil
}

func TestCheckBlockStatus_Blocked(t *testing.T) {
	blocked := map[string]bool{
		"user-a:user-b": true,
	}
	client, cleanup, err := setupTestClient(blocked)
	if err != nil {
		t.Fatalf("failed to setup test client: %v", err)
	}
	defer cleanup()

	isBlocked, err := client.CheckBlockStatus(context.Background(), "user-a", "user-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isBlocked {
		t.Error("expected blocked=true, got false")
	}
}

func TestCheckBlockStatus_ReverseBlocked(t *testing.T) {
	blocked := map[string]bool{
		"user-b:user-a": true,
	}
	client, cleanup, err := setupTestClient(blocked)
	if err != nil {
		t.Fatalf("failed to setup test client: %v", err)
	}
	defer cleanup()

	isBlocked, err := client.CheckBlockStatus(context.Background(), "user-a", "user-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isBlocked {
		t.Error("expected blocked=true (reverse direction), got false")
	}
}

func TestCheckBlockStatus_NotBlocked(t *testing.T) {
	client, cleanup, err := setupTestClient(nil)
	if err != nil {
		t.Fatalf("failed to setup test client: %v", err)
	}
	defer cleanup()

	isBlocked, err := client.CheckBlockStatus(context.Background(), "user-a", "user-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isBlocked {
		t.Error("expected blocked=false, got true")
	}
}

func TestCheckBlockStatus_DifferentUsers(t *testing.T) {
	blocked := map[string]bool{
		"user-a:user-b": true,
	}
	client, cleanup, err := setupTestClient(blocked)
	if err != nil {
		t.Fatalf("failed to setup test client: %v", err)
	}
	defer cleanup()

	isBlocked, err := client.CheckBlockStatus(context.Background(), "user-c", "user-d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isBlocked {
		t.Error("expected blocked=false for different users, got true")
	}
}
