package get_grpc_view

type Input struct {
	ServiceName string
}

type Output struct {
	ServiceName string
	Services    []ServiceView
}

type ServiceView struct {
	Name    string
	Methods []MethodView
}

type MethodView struct {
	Name      string
	Input     MessageView
	Output    MessageView
	Consumers []string
}

type MessageView struct {
	Name   string
	Fields []FieldView
}

type FieldView struct {
	Name      string
	Type      string
	Number    int32
	Consumers []string
}
