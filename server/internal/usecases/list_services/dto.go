package list_services

type Output struct {
	Services []Service
}

type Service struct {
	ID   string
	Name string
}
