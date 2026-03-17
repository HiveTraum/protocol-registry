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

type BreakingChangesValidatorProtocompile struct{}

func NewBreakingChangesValidatorProtocompile() *BreakingChangesValidatorProtocompile {
	return &BreakingChangesValidatorProtocompile{}
}

func (v *BreakingChangesValidatorProtocompile) Validate(ctx context.Context, previous, current entities.ProtoFileSet) error {
	prevFile, err := v.compile(ctx, previous)
	if err != nil {
		return fmt.Errorf("compile previous: %w", err)
	}

	currFile, err := v.compile(ctx, current)
	if err != nil {
		return fmt.Errorf("compile current: %w", err)
	}

	var changes []entities.BreakingChange

	v.checkMessages(prevFile.Messages(), currFile.Messages(), &changes)
	v.checkEnums(prevFile.Enums(), currFile.Enums(), &changes)
	v.checkServices(prevFile.Services(), currFile.Services(), &changes)

	if len(changes) > 0 {
		return entities.NewBreakingChangesError(changes)
	}

	return nil
}

func (v *BreakingChangesValidatorProtocompile) compile(ctx context.Context, fileSet entities.ProtoFileSet) (protoreflect.FileDescriptor, error) {
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

func (v *BreakingChangesValidatorProtocompile) checkMessages(prev, curr protoreflect.MessageDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[string]protoreflect.MessageDescriptor)
	for i := 0; i < prev.Len(); i++ {
		msg := prev.Get(i)
		prevMap[string(msg.Name())] = msg
	}

	currMap := make(map[string]protoreflect.MessageDescriptor)
	for i := 0; i < curr.Len(); i++ {
		msg := curr.Get(i)
		currMap[string(msg.Name())] = msg
	}

	for name := range prevMap {
		if _, ok := currMap[name]; !ok {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeMessageRemoved,
				Subject: name,
				Message: fmt.Sprintf("message %q was removed", name),
			})
		}
	}

	for name, prevMsg := range prevMap {
		currMsg, ok := currMap[name]
		if !ok {
			continue
		}
		v.checkFields(name, prevMsg.Fields(), currMsg.Fields(), changes)
		v.checkMessages(prevMsg.Messages(), currMsg.Messages(), changes)
		v.checkEnums(prevMsg.Enums(), currMsg.Enums(), changes)
	}
}

func (v *BreakingChangesValidatorProtocompile) checkFields(msgName string, prev, curr protoreflect.FieldDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[protoreflect.FieldNumber]protoreflect.FieldDescriptor)
	for i := 0; i < prev.Len(); i++ {
		f := prev.Get(i)
		prevMap[f.Number()] = f
	}

	currMap := make(map[protoreflect.FieldNumber]protoreflect.FieldDescriptor)
	for i := 0; i < curr.Len(); i++ {
		f := curr.Get(i)
		currMap[f.Number()] = f
	}

	for num, prevField := range prevMap {
		currField, ok := currMap[num]
		fieldName := string(prevField.Name())
		subject := fmt.Sprintf("%s.%s", msgName, fieldName)

		if !ok {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeFieldRemoved,
				Subject: subject,
				Message: fmt.Sprintf("field %q (number %d) was removed from message %q", fieldName, num, msgName),
			})
			continue
		}

		if prevField.Kind() != currField.Kind() {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeFieldTypeChanged,
				Subject: subject,
				Message: fmt.Sprintf("field %q in message %q changed type from %s to %s", fieldName, msgName, prevField.Kind(), currField.Kind()),
			})
		}

		if prevField.Cardinality() != currField.Cardinality() {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeFieldCardinalityChanged,
				Subject: subject,
				Message: fmt.Sprintf("field %q in message %q changed cardinality from %s to %s", fieldName, msgName, prevField.Cardinality(), currField.Cardinality()),
			})
		}
	}

	for num, currField := range currMap {
		if prevField, ok := prevMap[num]; ok {
			if prevField.Name() != currField.Name() {
				*changes = append(*changes, entities.BreakingChange{
					Type:    entities.BreakingChangeFieldRenamed,
					Subject: fmt.Sprintf("%s.%d", msgName, num),
					Message: fmt.Sprintf("field number %d in message %q was renamed from %q to %q", num, msgName, prevField.Name(), currField.Name()),
				})
			}
		}
	}
}

func (v *BreakingChangesValidatorProtocompile) checkEnums(prev, curr protoreflect.EnumDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[string]protoreflect.EnumDescriptor)
	for i := 0; i < prev.Len(); i++ {
		e := prev.Get(i)
		prevMap[string(e.Name())] = e
	}

	currMap := make(map[string]protoreflect.EnumDescriptor)
	for i := 0; i < curr.Len(); i++ {
		e := curr.Get(i)
		currMap[string(e.Name())] = e
	}

	for name := range prevMap {
		if _, ok := currMap[name]; !ok {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeEnumRemoved,
				Subject: name,
				Message: fmt.Sprintf("enum %q was removed", name),
			})
		}
	}

	for name, prevEnum := range prevMap {
		currEnum, ok := currMap[name]
		if !ok {
			continue
		}
		v.checkEnumValues(name, prevEnum.Values(), currEnum.Values(), changes)
	}
}

func (v *BreakingChangesValidatorProtocompile) checkEnumValues(enumName string, prev, curr protoreflect.EnumValueDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[protoreflect.EnumNumber]protoreflect.EnumValueDescriptor)
	for i := 0; i < prev.Len(); i++ {
		val := prev.Get(i)
		prevMap[val.Number()] = val
	}

	for num, prevVal := range prevMap {
		found := false
		for i := 0; i < curr.Len(); i++ {
			if curr.Get(i).Number() == num {
				found = true
				break
			}
		}
		if !found {
			valName := string(prevVal.Name())
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeEnumValueRemoved,
				Subject: fmt.Sprintf("%s.%s", enumName, valName),
				Message: fmt.Sprintf("enum value %q (number %d) was removed from enum %q", valName, num, enumName),
			})
		}
	}
}

func (v *BreakingChangesValidatorProtocompile) checkServices(prev, curr protoreflect.ServiceDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[string]protoreflect.ServiceDescriptor)
	for i := 0; i < prev.Len(); i++ {
		svc := prev.Get(i)
		prevMap[string(svc.Name())] = svc
	}

	currMap := make(map[string]protoreflect.ServiceDescriptor)
	for i := 0; i < curr.Len(); i++ {
		svc := curr.Get(i)
		currMap[string(svc.Name())] = svc
	}

	for name := range prevMap {
		if _, ok := currMap[name]; !ok {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeServiceRemoved,
				Subject: name,
				Message: fmt.Sprintf("service %q was removed", name),
			})
		}
	}

	for name, prevSvc := range prevMap {
		currSvc, ok := currMap[name]
		if !ok {
			continue
		}
		v.checkMethods(name, prevSvc.Methods(), currSvc.Methods(), changes)
	}
}

func (v *BreakingChangesValidatorProtocompile) checkMethods(svcName string, prev, curr protoreflect.MethodDescriptors, changes *[]entities.BreakingChange) {
	prevMap := make(map[string]protoreflect.MethodDescriptor)
	for i := 0; i < prev.Len(); i++ {
		m := prev.Get(i)
		prevMap[string(m.Name())] = m
	}

	currMap := make(map[string]protoreflect.MethodDescriptor)
	for i := 0; i < curr.Len(); i++ {
		m := curr.Get(i)
		currMap[string(m.Name())] = m
	}

	for name := range prevMap {
		if _, ok := currMap[name]; !ok {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeMethodRemoved,
				Subject: fmt.Sprintf("%s.%s", svcName, name),
				Message: fmt.Sprintf("method %q was removed from service %q", name, svcName),
			})
		}
	}

	for name, prevMethod := range prevMap {
		currMethod, ok := currMap[name]
		if !ok {
			continue
		}

		subject := fmt.Sprintf("%s.%s", svcName, name)

		if prevMethod.Input().FullName() != currMethod.Input().FullName() {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeMethodInputChanged,
				Subject: subject,
				Message: fmt.Sprintf("method %q in service %q changed input type from %q to %q", name, svcName, prevMethod.Input().FullName(), currMethod.Input().FullName()),
			})
		}

		if prevMethod.Output().FullName() != currMethod.Output().FullName() {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeMethodOutputChanged,
				Subject: subject,
				Message: fmt.Sprintf("method %q in service %q changed output type from %q to %q", name, svcName, prevMethod.Output().FullName(), currMethod.Output().FullName()),
			})
		}

		if prevMethod.IsStreamingClient() != currMethod.IsStreamingClient() || prevMethod.IsStreamingServer() != currMethod.IsStreamingServer() {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeMethodStreamingChanged,
				Subject: subject,
				Message: fmt.Sprintf("method %q in service %q changed streaming mode", name, svcName),
			})
		}
	}
}
