package communication

type Usecase interface {
	CheckMessage() (bool, error)
	ForwardMessage(serviceId string) error
}
