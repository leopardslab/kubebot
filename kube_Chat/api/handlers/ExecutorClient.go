package handlers

import (
	"context"
	"errors"
	kubebot "github.com/leopardslab/kubebot/kube_Chat/api/proto"
	"github.com/leopardslab/kubebot/kube_Chat/entity"
	"google.golang.org/grpc"
	"log"
)

type ExecutorClient struct {
	conn *grpc.ClientConn
}

func NewExecutorClient(conn *grpc.ClientConn) *ExecutorClient {
	return &ExecutorClient{
		conn: conn,
	}
}

func (e *ExecutorClient) RunCommand(msg *entity.Message) (*entity.CommandResult, error) {
	if e.conn == nil {
		return nil, errors.New("Grpc conn is nil")
	}
	reqBody := &kubebot.ExecuteRequest{
		CommandStr: msg.Content,
	}

	c := kubebot.NewExecutorRoutesClient(e.conn)

	resp, err := c.ExecuteCommand(context.Background(), reqBody)
	if err != nil {
		log.Printf("Error whhile calling Executor service. Error : %v", err)
		return nil, err
	}

	cmdResult := entity.NewCommandResult(msg.Content, resp.Result, resp.Error)

	return cmdResult, nil
}
