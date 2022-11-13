package communication

import (
	"errors"
	"github.com/leopardslab/kubebot/kube_Chat/entity"
	"strings"
)

type Service struct {
	Msg *entity.Message
}

func NewService(msg *entity.Message) *Service {
	return &Service{
		Msg: msg,
	}

}

func (s *Service) CheckMessage() (bool, error) {
	// Msg.content should be the msg after bot's mentionId
	// @kubeBot run kubectl get pods. :: msg.content -> run kubectl get pods
	if s.Msg == nil || len(s.Msg.Content) <= 0 {
		words := strings.Fields(s.Msg.Content)
		if words[0] != "run" {
			return false, nil
		}
		return true, nil
	}
	return false, errors.New("Error : Message cannot nil")
}

func (s *Service) ForwardMessage(serviceId string) error {
	//TODO
	return nil
}
