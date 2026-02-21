package styler

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"reflect"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/ext"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// CELEnv encapsulates the CEL environment and provides methods to
// evaluate CEL expressions to generate labels based on the current
// state of Kubernetes resources.
type CELEnv struct {
	Environment *cel.Env
}

// NewCELEnv creates and initializes a new CELEnv with the necessary cel
// environment and custom functions for evaluating expressions.
func NewCELEnv() (*CELEnv, error) {
	celEnv := &CELEnv{}

	envOpts := []cel.EnvOption{
		cel.Variable("resources", cel.DynType),

		ext.Bindings(),
		ext.Encoders(),
		ext.Lists(),
		ext.Math(),
		ext.Protos(),
		ext.Sets(),
		ext.Strings(),

		ext.NativeTypes(reflect.TypeFor[*x509.Certificate]()),
		cel.Function("parseCert",
			cel.Overload(
				"parseCert_dyn",
				[]*cel.Type{cel.DynType},
				cel.DynType,
				cel.FunctionBinding(func(args ...ref.Val) ref.Val {
					var b []byte

					switch v := args[0].Value().(type) {
					case string:
						var err error

						b, err = base64.StdEncoding.DecodeString(v)
						if err != nil {
							return types.WrapErr(fmt.Errorf("base64 decode failed: %w", err))
						}
					case []byte:
						b = v
					default:
						return types.WrapErr(fmt.Errorf("unsupported type for parseCert: %T", v))
					}

					if block, _ := pem.Decode(b); block != nil {
						b = block.Bytes
					}

					cert, err := x509.ParseCertificate(b)
					if err != nil {
						return types.WrapErr(fmt.Errorf("parseCertificate failed: %w", err))
					}

					return celEnv.Environment.CELTypeAdapter().NativeToValue(cert)
				}),
			),
		),
	}

	var err error

	celEnv.Environment, err = cel.NewEnv(envOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create CEL environment: %w", err)
	}

	return celEnv, nil
}

func (celEnv *CELEnv) expandLabel(ctx context.Context, label string, resources []unstructured.Unstructured) (string, error) {
	ast, issues := celEnv.Environment.Compile(label)
	if issues.Err() != nil {
		return "", fmt.Errorf("failed to compile CEL expression %s: %w", label, issues.Err())
	}

	prg, err := celEnv.Environment.Program(ast)
	if err != nil {
		return "", fmt.Errorf("failed to create CEL program for expression %s: %w", label, err)
	}

	convertedResources := make([]map[string]any, len(resources))
	for i, resource := range resources {
		convertedResources[i] = resource.UnstructuredContent()
	}

	val, _, err := prg.ContextEval(ctx, map[string]any{"resources": convertedResources})
	if err != nil {
		return "", fmt.Errorf("failed to evaluate CEL expression %s: %w", label, err)
	}

	return val.Value().(string), nil
}
