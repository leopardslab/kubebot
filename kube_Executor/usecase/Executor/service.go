package Executor

import (
	"errors"
	"fmt"
	"github.com/leopardslab/kubebot/kube_Executor/entity"
	"strings"
)

type Service struct {
	cmdStr string
}

func NewExecutorService(cmdStr string) *Service {
	return &Service{cmdStr: cmdStr}
}

func (s *Service) RunCommand() (*entity.CommandResult, error) {
	var cmd entity.Command

	fmt.Printf("Service : %s", s.cmdStr)

	if len(s.cmdStr) == 0 {
		fmt.Printf("1233")
		return &entity.CommandResult{Cmd: s.cmdStr, Result: "", Error: "empty command received"}, errors.New("empty command received")
	}
	splitCmd := strings.Split(s.cmdStr, " ")
	if splitCmd[0] != "kubectl" {
		return &entity.CommandResult{Cmd: s.cmdStr, Result: "", Error: "kubectl command not found"}, errors.New("kubectl command not found")
	}
	cmd.Head = splitCmd[0]
	if len(splitCmd) == 1 {
		cmd.Arguments = nil
	} else {
		cmd.Arguments = splitCmd[1:]
	}

	out, err := cmd.Run()
	if err != nil {
		return nil, err
	}

	return &entity.CommandResult{Cmd: s.cmdStr, Result: out, Error: ""}, nil
}
