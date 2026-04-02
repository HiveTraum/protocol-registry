package implementations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/user/protocol_registry/internal/entities"
	"gopkg.in/yaml.v3"
)

type OpenAPISyntaxValidator struct{}

func NewOpenAPISyntaxValidator() *OpenAPISyntaxValidator {
	return &OpenAPISyntaxValidator{}
}

func (v *OpenAPISyntaxValidator) Validate(_ entities.ProtocolType, fileSet entities.ProtoFileSet) error {
	entryContent := v.findEntry(fileSet)
	if entryContent == nil {
		return entities.NewSyntaxError([]string{fmt.Sprintf("entry point %q not found in file set", fileSet.EntryPoint)})
	}

	spec, err := v.parseSpec(entryContent)
	if err != nil {
		return entities.NewSyntaxError([]string{fmt.Sprintf("failed to parse OpenAPI spec: %v", err)})
	}

	var errs []string

	// Validate required top-level fields
	openapiVersion, ok := spec["openapi"].(string)
	if !ok || openapiVersion == "" {
		errs = append(errs, "missing required field: openapi (must be a string, e.g. '3.0.0')")
	} else if !strings.HasPrefix(openapiVersion, "3.") {
		errs = append(errs, fmt.Sprintf("unsupported OpenAPI version %q: only OpenAPI 3.x is supported", openapiVersion))
	}

	if info, ok := spec["info"]; !ok || info == nil {
		errs = append(errs, "missing required field: info")
	} else if infoMap, ok := info.(map[string]interface{}); ok {
		if _, ok := infoMap["title"]; !ok {
			errs = append(errs, "info.title is required")
		}
		if _, ok := infoMap["version"]; !ok {
			errs = append(errs, "info.version is required")
		}
	}

	if paths, ok := spec["paths"]; !ok || paths == nil {
		errs = append(errs, "missing required field: paths")
	} else if pathsMap, ok := paths.(map[string]interface{}); ok {
		for path, pathItem := range pathsMap {
			if pathItem == nil {
				continue
			}
			pathItemMap, ok := pathItem.(map[string]interface{})
			if !ok {
				errs = append(errs, fmt.Sprintf("path %q: invalid path item format", path))
				continue
			}
			httpMethods := []string{"get", "post", "put", "patch", "delete", "head", "options", "trace"}
			for _, method := range httpMethods {
				opRaw, exists := pathItemMap[method]
				if !exists {
					continue
				}
				opMap, ok := opRaw.(map[string]interface{})
				if !ok {
					errs = append(errs, fmt.Sprintf("path %q, method %s: invalid operation format", path, method))
					continue
				}
				responses, hasResponses := opMap["responses"]
				if !hasResponses || responses == nil {
					errs = append(errs, fmt.Sprintf("path %q, method %s: missing required field 'responses'", path, method))
				}
			}
		}
	}

	if len(errs) > 0 {
		return entities.NewSyntaxError(errs)
	}
	return nil
}

func (v *OpenAPISyntaxValidator) findEntry(fileSet entities.ProtoFileSet) []byte {
	for _, f := range fileSet.Files {
		if f.Path == fileSet.EntryPoint {
			return f.Content
		}
	}
	return nil
}

func (v *OpenAPISyntaxValidator) parseSpec(content []byte) (map[string]interface{}, error) {
	var result map[string]interface{}

	// Try JSON first
	if err := json.Unmarshal(content, &result); err == nil {
		return result, nil
	}

	// Fall back to YAML
	if err := yaml.Unmarshal(content, &result); err != nil {
		return nil, fmt.Errorf("content is neither valid JSON nor YAML: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("parsed spec is empty")
	}

	// YAML unmarshals maps as map[string]interface{} when possible,
	// but nested maps may come as map[interface{}]interface{} — normalize them.
	normalized, err := normalizeYAMLMap(result)
	if err != nil {
		return nil, fmt.Errorf("normalize yaml: %w", err)
	}
	return normalized, nil
}

// normalizeYAMLMap converts map[interface{}]interface{} (produced by yaml.v3 in some cases)
// to map[string]interface{} recursively.
func normalizeYAMLMap(v interface{}) (map[string]interface{}, error) {
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("expected map, got %T", v)
	}
	result := make(map[string]interface{}, len(m))
	for k, val := range m {
		result[k] = normalizeValue(val)
	}
	return result, nil
}

func normalizeValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(t))
		for k, val := range t {
			result[fmt.Sprintf("%v", k)] = normalizeValue(val)
		}
		return result
	case map[string]interface{}:
		result := make(map[string]interface{}, len(t))
		for k, val := range t {
			result[k] = normalizeValue(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(t))
		for i, val := range t {
			result[i] = normalizeValue(val)
		}
		return result
	default:
		return v
	}
}
