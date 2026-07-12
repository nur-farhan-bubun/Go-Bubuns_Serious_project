package userclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ride-sharing/shared/proto/user"
)

const (
	defaultTimeout = 3 * time.Second
)

type Client struct {
	conn *grpc.ClientConn
	stub user.UserServiceClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024)),
	)
	if err != nil {
		return nil, fmt.Errorf("gRPC dial: %w", err)
	}
	return &Client{
		conn: conn,
		stub: user.NewUserServiceClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CheckBlockStatus(ctx context.Context, senderID, recID string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	resp, err := c.stub.CheckBlockStatus(ctx, &user.BlocKStatusCheckRequest{
		SenderId: senderID,
		RecId:    recID,
	})
	if err != nil {
		return false, fmt.Errorf("check block status: %w", err)
	}
	return resp.IsBlocked, nil
}
