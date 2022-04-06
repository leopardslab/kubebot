package Executor

import "github.com/leopardslab/kubebot/kube_Executor/entity"

type Usecase interface {
	RunCommand() (*entity.CommandResult, error)
}
