package handlers

import (
	"context"
	"errors"
	"fmt"
	kubebot_executor "github.com/leopardslab/kubebot/kube_Executor/api/proto"
	"github.com/leopardslab/kubebot/kube_Executor/usecase/Executor"
	"log"
)

type ExecutorServer struct {
	kubebot_executor.UnsafeExecutorRoutesServer
}

func NewExecutorServer() kubebot_executor.ExecutorRoutesServer {
	return &ExecutorServer{}
}

var LOG string = "ExecutorServer"

func (e *ExecutorServer) ExecuteCommand(ctx context.Context, req *kubebot_executor.ExecuteRequest) (*kubebot_executor.ExecuteResponse, error) {
	log.Printf("%s: Executing Command", LOG)
	if req == nil {
		return nil, errors.New("req is nil")
	}

	fmt.Printf("%v", req)

	exeServ := Executor.NewExecutorService(req.CommandStr)
	result, err := exeServ.RunCommand()
	if err != nil {
		return &kubebot_executor.ExecuteResponse{
			Result: "",
			Error:  err.Error(),
		}, err
	}

	fmt.Printf("Result server : \n %s", result.Result)
	return &kubebot_executor.ExecuteResponse{
		Result: result.Result,
		Error:  result.Error,
	}, nil

}
