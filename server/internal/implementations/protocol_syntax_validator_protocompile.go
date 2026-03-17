package implementations

import (
	"context"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/user/protocol_registry/internal/entities"
)

type ProtocolSyntaxValidatorProtocompile struct{}

func NewProtocolSyntaxValidatorProtocompile() *ProtocolSyntaxValidatorProtocompile {
	return &ProtocolSyntaxValidatorProtocompile{}
}

func (v *ProtocolSyntaxValidatorProtocompile) Validate(fileSet entities.ProtoFileSet) error {
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

	_, err := compiler.Compile(context.Background(), fileSet.EntryPoint)
	if err != nil || len(errs) > 0 {
		if len(errs) > 0 {
			return entities.NewSyntaxError(errs)
		}
		return entities.NewSyntaxError([]string{err.Error()})
	}

	return nil
}
