package gojsonrpc2

import (
	"context"
	"encoding/json"
	"io"
)

type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`

	writer io.Writer
}

type JSONRPCResult struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func SetMessageWriter(m *JSONRPCMessage, w io.Writer) {
	m.writer = w
}

func (r *JSONRPCMessage) IsEvent() bool {
	return r.ID == nil
}

func (r *JSONRPCMessage) Respond(ctx context.Context, result any) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.RespondRaw(ctx, resultData)
}

func (r *JSONRPCMessage) RespondRaw(ctx context.Context, result json.RawMessage) error {
	m, err := json.Marshal(JSONRPCResult{
		JSONRPC: "2.0",
		ID:      r.ID,
		Result:  result,
	})
	if err != nil {
		return err
	}

	if _, err := r.writer.Write(m); err != nil {
		return err
	}
	return nil
}

func (r *JSONRPCMessage) RespondErr(ctx context.Context, code int, message string, data any) error {
	errRes := JSONRPCResult{
		JSONRPC: "2.0",
		ID:      r.ID,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
	}

	if data != nil {
		dataRaw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		errRes.Error.Data = dataRaw
	}

	m, err := json.Marshal(errRes)
	if err != nil {
		return err
	}

	if _, err := r.writer.Write(m); err != nil {
		return err
	}
	return nil
}

func BindParams[T any](ctx context.Context, r *JSONRPCMessage) (T, error) {
	var v T
	if err := json.Unmarshal(r.Params, &v); err != nil {
		_ = r.RespondErr(ctx, -32602, "Invalid params", nil)
		return v, err
	}
	return v, nil
}
