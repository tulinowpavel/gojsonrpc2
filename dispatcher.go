package gojsonrpc2

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sync"

	"github.com/invopop/jsonschema"
)

// JSONRPCDispatcher is server method/notification call dispatcher
type JSONRPCDispatcher struct {
	methods map[string]*Method
}

func NewJSONRPCDispatcher() *JSONRPCDispatcher {
	return &JSONRPCDispatcher{
		methods: make(map[string]*Method),
	}
}

type Method struct {
	docs    string
	errors  map[int]string
	handler MethodHandler
}

func (m *Method) ServeJSONRPC(ctx context.Context, r *JSONRPCRequest) {
	m.handler.handler(ctx, r)
}

func (d *JSONRPCDispatcher) Method(name string) *MethodBuilder {
	return &MethodBuilder{
		dispatcher: d,
		name:       name,
	}
}

var messagePool = sync.Pool{
	New: func() any {
		return &JSONRPCMessage{}
	},
}

func (d *JSONRPCDispatcher) ServeRaw(ctx context.Context, w io.Writer, msg json.RawMessage) error {
	m := messagePool.Get().(*JSONRPCMessage)
	defer messagePool.Put(m)

	if err := json.Unmarshal(msg, m); err != nil {
		// TODO: prepare error as bytes on init
		errMsg, _ := json.Marshal(JSONRPCResult{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32600,
				Message: "Invalid Request",
			},
		})

		if _, err := w.Write(errMsg); err != nil {
			return err
		}

		return nil
	}

	return d.Serve(ctx, w, m)
}

var requestPool = sync.Pool{
	New: func() any {
		return &JSONRPCRequest{}
	},
}

func (d *JSONRPCDispatcher) Serve(ctx context.Context, w io.Writer, msg *JSONRPCMessage) error {

	if method, ok := d.methods[msg.Method]; ok {
		// TODO: add request pool here
		r := requestPool.Get().(*JSONRPCRequest)
		r.Msg = msg
		r.writer = w
		defer requestPool.Put(r)

		method.ServeJSONRPC(ctx, r)
	} else {
		// TODO: prepare error as bytes on init
		errMsg, _ := json.Marshal(JSONRPCResult{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		})

		if _, err := w.Write(errMsg); err != nil {
			return err
		}
	}

	return nil
}

type MethodHandlerFunc func(ctx context.Context, r *JSONRPCRequest)

type MethodHandler struct {
	params  *jsonschema.Schema
	result  *jsonschema.Schema
	handler MethodHandlerFunc
}

type MethodBuilder struct {
	dispatcher *JSONRPCDispatcher
	name       string
	docs       string
	errors     map[int]string
	handler    MethodHandler
}

func (b *MethodBuilder) SetDocs(docs string) *MethodBuilder {
	b.docs = docs
	return b
}

func (b *MethodBuilder) DefineError(code int, description string) *MethodBuilder {
	b.errors[code] = description
	return b
}

func (b *MethodBuilder) SetHandler(h MethodHandler) *MethodBuilder {
	b.handler = h
	return b
}

func (b *MethodBuilder) SetHandlerFunc(h MethodHandlerFunc) *MethodBuilder {
	b.handler = MethodHandler{
		handler: h,
	}
	return b
}

func (b *MethodBuilder) Register() {
	b.dispatcher.methods[b.name] = &Method{
		docs:    b.docs,
		errors:  b.errors,
		handler: b.handler,
	}
}

func NewTypedHandler[TParams, TResult any](h func(ctx context.Context, params TParams) (TResult, error)) MethodHandler {
	return MethodHandler{
		params: GenSchema[TParams](), // TODO: reflect to get params type
		result: GenSchema[TResult](), // TODO: reflect to get result type
		handler: func(ctx context.Context, r *JSONRPCRequest) {
			params, err := BindParams[TParams](ctx, r)
			if err != nil {
				return
			}

			result, err := h(ctx, params)
			if err != nil {
				// TODO: if err is jsonrpc, then return it

				_ = r.RespondErr(ctx, -32099, "unknown application error", map[string]any{
					"error": err.Error(),
				})
				return
			}

			_ = r.Respond(ctx, result)
		},
	}
}

func GenSchema[T any]() *jsonschema.Schema {
	t := reflect.TypeOf((*T)(nil)).Elem()
	return jsonschema.ReflectFromType(t)
}
