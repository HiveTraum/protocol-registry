package validate_protocol

type ProtoFile struct {
	Path    string
	Content []byte
}

type Input struct {
	ServiceName     string
	ProtocolType    string
	ProtoDir        string
	EntryPoint      string
	AgainstVersions []string
}

type ConsumerViolation struct {
	ConsumerName string
	Violations   []string
}

type VersionViolation struct {
	Version   string
	Consumers []ConsumerViolation
}

type Output struct {
	Valid      bool
	Violations []VersionViolation
}
