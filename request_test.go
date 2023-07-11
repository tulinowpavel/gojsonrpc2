package gojsonrpc2_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/tulinowpavel/gojsonrpc2"
)

func TestRequest__Respond(t *testing.T) {

	buf := bytes.Buffer{}

	req := gojsonrpc2.NewJSONRPCRequest(
		&gojsonrpc2.JSONRPCMessage{
			ID:     1,
			Method: "banana",
			Params: json.RawMessage(`{"message": "hello banana"}`),
		},
		&buf,
	)

	if req.IsEvent() {
		t.Error("request is not event but marked as event")
	}

	if err := req.Respond(context.Background(), map[string]any{
		"banana": "forever",
	}); err != nil {
		t.Fatalf("unexpected respond error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal result message error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"result": map[string]any{
			"banana": "forever",
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result message not equals to reference: reference=%#v actual=%#v", refRes, res)
	}

}

func TestRequest__RespondWithError(t *testing.T) {

	buf := bytes.Buffer{}

	req := gojsonrpc2.NewJSONRPCRequest(&gojsonrpc2.JSONRPCMessage{
		ID:     1,
		Method: "banana",
		Params: json.RawMessage(`{"message": "hello banana"}`),
	}, &buf)

	if err := req.RespondErr(context.Background(), 1000, "some error", map[string]any{
		"banana": "forever",
	}); err != nil {
		t.Fatalf("unexpected respond error: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal result message error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"error": map[string]any{
			"code":    float64(1000),
			"message": "some error",
			"data": map[string]any{
				"banana": "forever",
			},
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result message not equals to reference: reference=%#v actual=%#v", refRes, res)
	}
}

func TestRequest__MustRespondWithErrorOnBindParamsError(t *testing.T) {
	buf := bytes.Buffer{}

	req := gojsonrpc2.NewJSONRPCRequest(&gojsonrpc2.JSONRPCMessage{
		ID:     1,
		Method: "banana",
		Params: json.RawMessage(`{"message", "hello banana"]`),
	}, &buf)

	if _, err := gojsonrpc2.BindParams[any](context.Background(), req); err == nil {
		t.Fatalf("unexpected success. must fail")
	}

	var res map[string]any
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal result message error: %v", err)
	}

	refRes := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"error": map[string]any{
			"code":    float64(-32602),
			"message": "Invalid params",
		},
	}

	if !reflect.DeepEqual(res, refRes) {
		t.Fatalf("result message not equals to reference: reference=%#v actual=%#v", refRes, res)
	}
}
