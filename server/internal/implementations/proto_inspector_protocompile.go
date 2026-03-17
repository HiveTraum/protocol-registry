package implementations

import (
	"context"
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/user/protocol_registry/internal/entities"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type ProtoInspectorProtocompile struct{}

func NewProtoInspectorProtocompile() *ProtoInspectorProtocompile {
	return &ProtoInspectorProtocompile{}
}

func (p *ProtoInspectorProtocompile) Inspect(ctx context.Context, fileSet entities.ProtoFileSet) ([]entities.ProtoServiceView, error) {
	fileDesc, err := p.compile(ctx, fileSet)
	if err != nil {
		return nil, fmt.Errorf("compile proto: %w", err)
	}

	services := fileDesc.Services()
	result := make([]entities.ProtoServiceView, 0, services.Len())

	for i := 0; i < services.Len(); i++ {
		svc := services.Get(i)
		result = append(result, p.inspectService(svc))
	}

	return result, nil
}

func (p *ProtoInspectorProtocompile) compile(ctx context.Context, fileSet entities.ProtoFileSet) (protoreflect.FileDescriptor, error) {
	var errs []string
	rep := reporter.NewReporter(
		func(err reporter.ErrorWithPos) error {
			errs = append(errs, err.Error())
			return nil
		},
		nil,
	)

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(fileSet.ToSourceMap()),
		},
		Reporter: rep,
	}

	files, err := compiler.Compile(ctx, fileSet.EntryPoint)
	if err != nil || len(errs) > 0 {
		if len(errs) > 0 {
			return nil, fmt.Errorf("parse errors: %s", strings.Join(errs, "; "))
		}
		return nil, err
	}

	return files[0], nil
}

func (p *ProtoInspectorProtocompile) inspectService(svc protoreflect.ServiceDescriptor) entities.ProtoServiceView {
	methods := svc.Methods()
	views := make([]entities.ProtoMethodView, 0, methods.Len())

	for i := 0; i < methods.Len(); i++ {
		m := methods.Get(i)
		views = append(views, entities.ProtoMethodView{
			Name:   string(m.Name()),
			Input:  p.inspectMessage(m.Input()),
			Output: p.inspectMessage(m.Output()),
		})
	}

	return entities.ProtoServiceView{
		Name:    string(svc.Name()),
		Methods: views,
	}
}

func (p *ProtoInspectorProtocompile) inspectMessage(msg protoreflect.MessageDescriptor) entities.ProtoMessageView {
	fields := msg.Fields()
	views := make([]entities.ProtoFieldView, 0, fields.Len())

	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		views = append(views, entities.ProtoFieldView{
			Name:   string(f.Name()),
			Type:   fieldType(f),
			Number: int32(f.Number()),
		})
	}

	return entities.ProtoMessageView{
		Name:   string(msg.Name()),
		Fields: views,
	}
}

func fieldType(f protoreflect.FieldDescriptor) string {
	if f.IsMap() {
		keyType := scalarOrMessageName(f.MapKey())
		valType := scalarOrMessageName(f.MapValue())
		return fmt.Sprintf("map<%s, %s>", keyType, valType)
	}

	base := scalarOrMessageName(f)

	if f.IsList() {
		return "repeated " + base
	}

	return base
}

func scalarOrMessageName(f protoreflect.FieldDescriptor) string {
	switch f.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return string(f.Message().Name())
	case protoreflect.EnumKind:
		return string(f.Enum().Name())
	default:
		return f.Kind().String()
	}
}
