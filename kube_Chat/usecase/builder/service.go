package builder

import (
	"github.com/leopardslab/kubebot/kube_Chat/entity"
)

type Service struct {
	cmdCtx *entity.CommandContext
}

func (s *Service) BuildCommand() (*entity.CommandContext, error) {
	//This function checks if the command detected by dialog flow req some argument or not if req then sends response as
	//query to user and if its complete then to the executor

	//TODO  Add logic for building commands

	return s.cmdCtx, nil
}
