package dialogflow

import "github.com/leopardslab/kubebot/kube_Chat/entity"

type Usecase interface {
	DetectCommand() (*entity.CommandContext, error)
}
