package control

type Service struct {
	pending map[uint16]func(error)
	// TODO
}

func (s *Service) Register(req map[string]interface{}, _ *struct{}) (err error) {
	return // TODO
}
