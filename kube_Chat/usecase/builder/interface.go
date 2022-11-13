package builder

import "github.com/leopardslab/kubebot/kube_Chat/entity"

type Usecase interface {
	BuildCommand() (*entity.Command, error)
}
