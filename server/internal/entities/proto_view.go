package entities

type ProtoServiceView struct {
	Name    string
	Methods []ProtoMethodView
}

type ProtoMethodView struct {
	Name   string
	Input  ProtoMessageView
	Output ProtoMessageView
}

type ProtoMessageView struct {
	Name   string
	Fields []ProtoFieldView
}

type ProtoFieldView struct {
	Name   string
	Type   string
	Number int32
}
