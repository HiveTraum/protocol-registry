package implementations

import (
	"context"
	"fmt"

	"github.com/user/protocol_registry/internal/entities"
)

// BreakingChangesValidatorOpenAPI detects breaking changes between two OpenAPI 3.x specs.
//
// Breaking changes detected:
//   - Removed endpoints (paths)
//   - Removed HTTP methods from existing endpoints
//   - Removed path parameters
//   - Removed required request body fields or type changes
//   - Removed required response fields or type changes (2xx responses)
type BreakingChangesValidatorOpenAPI struct {
	syntaxHelper *OpenAPISyntaxValidator
}

func NewBreakingChangesValidatorOpenAPI() *BreakingChangesValidatorOpenAPI {
	return &BreakingChangesValidatorOpenAPI{
		syntaxHelper: NewOpenAPISyntaxValidator(),
	}
}

func (v *BreakingChangesValidatorOpenAPI) Validate(ctx context.Context, _ entities.ProtocolType, previous, current entities.ProtoFileSet) error {
	prevSpec, err := v.parseAndResolve(previous)
	if err != nil {
		return fmt.Errorf("parse previous spec: %w", err)
	}
	currSpec, err := v.parseAndResolve(current)
	if err != nil {
		return fmt.Errorf("parse current spec: %w", err)
	}

	var changes []entities.BreakingChange

	prevPaths := getMapField(prevSpec, "paths")
	currPaths := getMapField(currSpec, "paths")

	prevComponents := getMapField(prevSpec, "components")
	currComponents := getMapField(currSpec, "components")

	for path, prevItem := range prevPaths {
		currItem, exists := currPaths[path]
		if !exists {
			changes = append(changes, entities.BreakingChange{
				Type:    entities.BreakingChangeEndpointRemoved,
				Subject: path,
				Message: fmt.Sprintf("endpoint %q was removed", path),
			})
			continue
		}
		prevItemMap := toMap(prevItem)
		currItemMap := toMap(currItem)
		v.checkPathItem(path, prevItemMap, currItemMap, prevComponents, currComponents, prevSpec, currSpec, &changes)
	}

	if len(changes) > 0 {
		return entities.NewBreakingChangesError(changes)
	}
	return nil
}

func (v *BreakingChangesValidatorOpenAPI) parseAndResolve(fileSet entities.ProtoFileSet) (map[string]interface{}, error) {
	content := v.syntaxHelper.findEntry(fileSet)
	if content == nil {
		return nil, fmt.Errorf("entry point %q not found in file set", fileSet.EntryPoint)
	}
	return v.syntaxHelper.parseSpec(content)
}

func (v *BreakingChangesValidatorOpenAPI) checkPathItem(
	path string,
	prev, curr map[string]interface{},
	prevComponents, currComponents map[string]interface{},
	prevSpec, currSpec map[string]interface{},
	changes *[]entities.BreakingChange,
) {
	httpMethods := []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}
	for _, method := range httpMethods {
		prevOp, prevHas := prev[method]
		currOp, currHas := curr[method]

		if prevHas && !currHas {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangeHTTPMethodRemoved,
				Subject: fmt.Sprintf("%s %s", method, path),
				Message: fmt.Sprintf("HTTP method %s was removed from endpoint %q", method, path),
			})
			continue
		}

		if !prevHas {
			continue
		}

		prevOpMap := toMap(prevOp)
		currOpMap := toMap(currOp)

		subject := fmt.Sprintf("%s %s", method, path)

		// Check path parameters
		v.checkParameters(subject, prevOpMap, currOpMap, changes)

		// Check request body
		v.checkRequestBody(subject, prevOpMap, currOpMap, prevComponents, currComponents, prevSpec, currSpec, changes)

		// Check responses
		v.checkResponses(subject, prevOpMap, currOpMap, prevComponents, currComponents, prevSpec, currSpec, changes)
	}
}

func (v *BreakingChangesValidatorOpenAPI) checkParameters(
	subject string,
	prevOp, currOp map[string]interface{},
	changes *[]entities.BreakingChange,
) {
	prevParams := v.collectParameters(prevOp)
	currParams := v.collectParameters(currOp)

	for name, prevParam := range prevParams {
		if _, exists := currParams[name]; !exists {
			*changes = append(*changes, entities.BreakingChange{
				Type:    entities.BreakingChangePathParameterRemoved,
				Subject: fmt.Sprintf("%s/parameters/%s", subject, name),
				Message: fmt.Sprintf("parameter %q was removed from %s", name, subject),
			})
		}
		_ = prevParam
	}
}

// collectParameters builds a map of paramName -> param for an operation.
func (v *BreakingChangesValidatorOpenAPI) collectParameters(op map[string]interface{}) map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})
	paramsRaw, ok := op["parameters"]
	if !ok {
		return result
	}
	params, ok := paramsRaw.([]interface{})
	if !ok {
		return result
	}
	for _, p := range params {
		pm := toMap(p)
		if pm == nil {
			continue
		}
		name, _ := pm["name"].(string)
		if name != "" {
			result[name] = pm
		}
	}
	return result
}

func (v *BreakingChangesValidatorOpenAPI) checkRequestBody(
	subject string,
	prevOp, currOp map[string]interface{},
	prevComponents, currComponents map[string]interface{},
	prevSpec, currSpec map[string]interface{},
	changes *[]entities.BreakingChange,
) {
	prevBody, hasPrevBody := prevOp["requestBody"]
	currBody, hasCurrBody := currOp["requestBody"]

	if !hasPrevBody {
		return
	}
	if !hasCurrBody {
		// request body removed entirely — breaking
		*changes = append(*changes, entities.BreakingChange{
			Type:    entities.BreakingChangeRequestFieldRemoved,
			Subject: subject + "/requestBody",
			Message: fmt.Sprintf("requestBody was removed from %s", subject),
		})
		return
	}

	prevSchema := v.extractSchemaFromContent(toMap(prevBody), prevComponents, prevSpec)
	currSchema := v.extractSchemaFromContent(toMap(currBody), currComponents, currSpec)

	if prevSchema != nil && currSchema != nil {
		v.checkSchema(subject+"/requestBody", prevSchema, currSchema,
			entities.BreakingChangeRequestFieldRemoved, entities.BreakingChangeRequestFieldTypeChanged, changes)
	}
}

func (v *BreakingChangesValidatorOpenAPI) checkResponses(
	subject string,
	prevOp, currOp map[string]interface{},
	prevComponents, currComponents map[string]interface{},
	prevSpec, currSpec map[string]interface{},
	changes *[]entities.BreakingChange,
) {
	prevResponses := getMapField(prevOp, "responses")
	currResponses := getMapField(currOp, "responses")

	// Only check 2xx response codes
	for code, prevResp := range prevResponses {
		if !is2xx(code) {
			continue
		}
		currResp, exists := currResponses[code]
		if !exists {
			continue // response code removal is not always breaking (could be an addition)
		}

		prevSchema := v.extractSchemaFromContent(toMap(prevResp), prevComponents, prevSpec)
		currSchema := v.extractSchemaFromContent(toMap(currResp), currComponents, currSpec)

		if prevSchema != nil && currSchema != nil {
			loc := fmt.Sprintf("%s/responses/%s", subject, code)
			v.checkSchema(loc, prevSchema, currSchema,
				entities.BreakingChangeResponseFieldRemoved, entities.BreakingChangeResponseFieldTypeChanged, changes)
		}
	}
}

// extractSchemaFromContent returns the first JSON application/json schema from
// a requestBody or response object, resolving $ref if needed.
func (v *BreakingChangesValidatorOpenAPI) extractSchemaFromContent(
	obj map[string]interface{},
	components map[string]interface{},
	spec map[string]interface{},
) map[string]interface{} {
	if obj == nil {
		return nil
	}
	content := getMapField(obj, "content")
	for _, mediaType := range content {
		mt := toMap(mediaType)
		if mt == nil {
			continue
		}
		schema := toMap(mt["schema"])
		if schema != nil {
			return v.resolveRef(schema, components, spec)
		}
	}
	return nil
}

// resolveRef follows a $ref pointer within the spec if present.
func (v *BreakingChangesValidatorOpenAPI) resolveRef(
	schema map[string]interface{},
	components map[string]interface{},
	spec map[string]interface{},
) map[string]interface{} {
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	// Only handle internal refs of the form #/components/schemas/Foo
	const prefix = "#/components/schemas/"
	if len(ref) <= len(prefix) {
		return schema
	}
	if ref[:len(prefix)] != prefix {
		return schema
	}
	schemaName := ref[len(prefix):]
	schemas := getMapField(components, "schemas")
	if s, ok := schemas[schemaName]; ok {
		resolved := toMap(s)
		if resolved != nil {
			return resolved
		}
	}
	return schema
}

// checkSchema compares two OpenAPI schema objects and appends breaking changes.
func (v *BreakingChangesValidatorOpenAPI) checkSchema(
	location string,
	prev, curr map[string]interface{},
	removedType, typeChangedType entities.BreakingChangeType,
	changes *[]entities.BreakingChange,
) {
	prevProps := getMapField(prev, "properties")
	currProps := getMapField(curr, "properties")

	prevRequired := toStringSet(prev["required"])
	currRequired := toStringSet(curr["required"])

	for fieldName, prevPropRaw := range prevProps {
		prevProp := toMap(prevPropRaw)

		if _, exists := currProps[fieldName]; !exists {
			// Field entirely removed
			if prevRequired[fieldName] {
				*changes = append(*changes, entities.BreakingChange{
					Type:    removedType,
					Subject: fmt.Sprintf("%s/%s", location, fieldName),
					Message: fmt.Sprintf("required field %q was removed from %s", fieldName, location),
				})
			}
			continue
		}

		currProp := toMap(currProps[fieldName])
		if prevProp == nil || currProp == nil {
			continue
		}

		// Check type change
		prevType, _ := prevProp["type"].(string)
		currType, _ := currProp["type"].(string)
		if prevType != "" && currType != "" && prevType != currType {
			*changes = append(*changes, entities.BreakingChange{
				Type:    typeChangedType,
				Subject: fmt.Sprintf("%s/%s", location, fieldName),
				Message: fmt.Sprintf("field %q in %s changed type from %q to %q", fieldName, location, prevType, currType),
			})
		}

		// Check if field became required in consumer but is no longer required in server
		// (not breaking) – we only care about removal of previously required fields.

		// Also check: field was required in previous, is no longer present in current required list
		if prevRequired[fieldName] && !currRequired[fieldName] {
			// Making a required field optional is not a breaking change for consumers.
		}
	}
}

// Helpers

func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	result, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return result
}

func toMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	m, _ := v.(map[string]interface{})
	return m
}

func toStringSet(v interface{}) map[string]bool {
	result := make(map[string]bool)
	arr, ok := v.([]interface{})
	if !ok {
		return result
	}
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result[s] = true
		}
	}
	return result
}

func is2xx(code string) bool {
	return len(code) == 3 && code[0] == '2'
}
