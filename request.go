package gojsonrpc2

import (
	"context"
	"encoding/json"
	"io"
)

type JSONRPCRequest struct {
	Msg    *JSONRPCMessage
	writer io.Writer
}

func NewJSONRPCRequest(msg *JSONRPCMessage, w io.Writer) *JSONRPCRequest {
	return &JSONRPCRequest{
		Msg:    msg,
		writer: w,
	}
}

func (r *JSONRPCRequest) IsEvent() bool {
	return r.Msg.ID == nil
}

func (r *JSONRPCRequest) Respond(ctx context.Context, result any) error {
	resultData, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return r.RespondRaw(ctx, resultData)
}

func (r *JSONRPCRequest) RespondRaw(ctx context.Context, result json.RawMessage) error {
	m, err := json.Marshal(JSONRPCResult{
		JSONRPC: "2.0",
		ID:      r.Msg.ID,
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

func (r *JSONRPCRequest) RespondErr(ctx context.Context, code int, message string, data any) error {
	errRes := JSONRPCResult{
		JSONRPC: "2.0",
		ID:      r.Msg.ID,
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

func BindParams[T any](ctx context.Context, r *JSONRPCRequest) (T, error) {
	var v T
	if err := json.Unmarshal(r.Msg.Params, &v); err != nil {
		_ = r.RespondErr(ctx, -32602, "Invalid params", nil)
		return v, err
	}
	return v, nil
}
