package gojsonrpc2_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/tulinowpavel/gojsonrpc2"
)

func TestDispatcher__MethodMustBeCalled(t *testing.T) {
	d := gojsonrpc2.NewJSONRPCDispatcher()

	d.Method("make_banana").SetHandlerFunc(func(ctx context.Context, r *gojsonrpc2.JSONRPCMessage) {
		t.Log("method called")
		_ = r.Respond(ctx, map[string]any{
			"banana": "forever",
		})
	}).Register()

	buf := bytes.Buffer{}

	if err := d.ServeRaw(context.Background(), &buf, json.RawMessage(`{"jsonrpc": "2.0", "id": 1, "method": "make_banana"}`)); err != nil {
		t.Fatalf("unexpected serve error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal result error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"result": map[string]any{
			"banana": "forever",
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result not equals to reference: reference=%#v result=%#v", refRes, res)
	}
}

func TestDispatcher__MustRespondWithErrorIfMethodNotFound(t *testing.T) {
	d := gojsonrpc2.NewJSONRPCDispatcher()

	buf := bytes.Buffer{}

	if err := d.ServeRaw(context.Background(), &buf, json.RawMessage(`{"jsonrpc": "2.0", "id": 1, "method": "make_banana"}`)); err != nil {
		t.Fatalf("unexpected serve error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"error": map[string]any{
			"code":    float64(-32601),
			"message": "Method not found",
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result not equals to reference: reference=%#v result=%#v", refRes, res)
	}
}

func TestDispatcher__MustRespondWithErrorOnInvalidPayload(t *testing.T) {
	d := gojsonrpc2.NewJSONRPCDispatcher()

	buf := bytes.Buffer{}

	if err := d.ServeRaw(context.Background(), &buf, json.RawMessage(`{"jsonrpc":: "2.0", "id": 1 "method": "make_banana"]`)); err != nil {
		t.Fatalf("unexpected serve error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"error": map[string]any{
			"code":    float64(-32600),
			"message": "Invalid Request",
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result not equals to reference: reference=%#v result=%#v", refRes, res)
	}
}
