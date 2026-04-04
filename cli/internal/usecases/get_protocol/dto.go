package get_protocol

type ProtoFile struct {
	Path    string
	Content []byte
}

type Input struct {
	ServiceName  string
	ProtocolType string
	Version      string
}

type Output struct {
	ServiceName string
	EntryPoint  string
	Files       []ProtoFile
}
