package publish_protocol

type ProtoFile struct {
	Path    string
	Content []byte
}

type Input struct {
	ServiceName  string
	ProtocolType string
	ProtoDir     string
	EntryPoint   string
	Versions     []string
}

type Output struct {
	ServiceName string
	IsNew       bool
}
