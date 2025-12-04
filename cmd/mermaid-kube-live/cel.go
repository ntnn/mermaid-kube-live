package main

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

	mkl "github.com/ntnn/mermaid-kube-live/pkg/mermaid-kube-live"
)

var env *cel.Env

func init() {
	envOpts := []cel.EnvOption{
		ext.NativeTypes(reflect.TypeOf(&mkl.ResourceState{})),
		cel.Variable("rs", cel.DynType),

		ext.Bindings(),
		ext.Encoders(),
		ext.Lists(),
		ext.Math(),
		ext.Protos(),
		ext.Sets(),
		ext.Strings(),

		ext.NativeTypes(reflect.TypeOf(&x509.Certificate{})),
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
					return env.CELTypeAdapter().NativeToValue(cert)
				}),
			),
		),
	}

	// TODO place in a once
	var err error
	env, err = cel.NewEnv(envOpts...)
	if err != nil {
		panic(fmt.Sprintf("failed to create CEL environment: %v", err))
	}
}

func expandLabel(ctx context.Context, label string, resourceState mkl.ResourceState) (string, error) {
	ast, issues := env.Compile(label)
	if issues.Err() != nil {
		return "", fmt.Errorf("failed to compile CEL expression %s: %w", label, issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return "", fmt.Errorf("failed to create CEL program for expression %s: %w", label, err)
	}

	val, _, err := prg.ContextEval(ctx, map[string]any{"rs": resourceState})
	if err != nil {
		return "", fmt.Errorf("failed to evaluate CEL expression %s: %w", label, err)
	}

	return val.Value().(string), nil
}
