package handlers

import (
	"context"
	"errors"
	kubebot "github.com/leopardslab/kubebot/kube_Chat/api/proto"
	"github.com/leopardslab/kubebot/kube_Chat/entity"
	"github.com/leopardslab/kubebot/kube_Chat/usecase/communication"
	"google.golang.org/grpc"
	"strings"
)

type MessageServer struct {
	kubebot.UnsafeMessageChatServerServer
}

// GetMessageResponse This function is called by Bot to get response for the users message.
func (m *MessageServer) GetMessageResponse(ctx context.Context, req *kubebot.BotCommandRequest) (*kubebot.BotCommandResponse, error) {
	var mentions []string
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if len(req.Mentions) > 0 {
		mentions = strings.Fields(req.Mentions)
	}

	msg := entity.NewMessage(req.Platform, req.Sender, req.Content, req.BotId, mentions)
	serv := communication.NewService(msg)
	isCommand, err := serv.CheckMessage()

	if err != nil {
		return nil, err
	}

	if isCommand == false {
		//TODO Handle dialog flow which will call builder if builder requires some argument this function will return query
	}

	var conn *grpc.ClientConn
	//TODO Add address for GRPC
	conn, err = grpc.Dial("address", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	handler := NewExecutorClient(conn)
	result, err := handler.RunCommand(msg)
	if err != nil {
		return nil, err
	}

	return &kubebot.BotCommandResponse{
		Response: result.Result,
		IsQuery:  false,
		Query:    "",
		Command:  result.Cmd,
	}, nil
}
