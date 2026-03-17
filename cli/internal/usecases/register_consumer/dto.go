package register_consumer

type ProtoFile struct {
	Path    string
	Content []byte
}

type Input struct {
	ConsumerName string
	ServerName   string
	ProtocolType string
	ProtoDir     string
	EntryPoint   string
}

type Output struct {
	ConsumerName string
	ServerName   string
	IsNew        bool
}
