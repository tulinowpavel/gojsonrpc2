package gojsonrpc2

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"sync/atomic"
)

type JSONRPCDuplexConnection struct {
	dispatcher *JSONRPCDispatcher

	lastId  atomic.Int32
	pending map[int32]chan<- *JSONRPCMessage
	l       sync.Mutex
	w       io.Writer
}

func NewJSONRPCDuplexConnection(w io.Writer, dispatcher *JSONRPCDispatcher) *JSONRPCDuplexConnection {
	return &JSONRPCDuplexConnection{
		dispatcher: dispatcher,
		pending:    make(map[int32]chan<- *JSONRPCMessage),
	}
}

func (c *JSONRPCDuplexConnection) ServeRaw(ctx context.Context, msg json.RawMessage) error {
	panic("not implemented")
}

func (c *JSONRPCDuplexConnection) Serve(ctx context.Context, msg *JSONRPCMessage) error {

	// handle reply
	if msg.Result != nil || msg.Error != nil {
		if msg.ID != nil {
			// by default json.Unmarshal parse numbers as float64.
			// string id may be ignored cause we send only int32 ids
			if id, ok := msg.ID.(float64); ok {
				c.l.Lock()
				if replyChan, ok := c.pending[int32(id)]; ok {
					replyChan <- msg
				}
				c.l.Unlock()
			}
		}

		return nil
	}

	// handle request
	if msg.Params != nil && c.dispatcher != nil {
		return c.dispatcher.Serve(ctx, c.w, msg)
	}

	return nil
}

func (c *JSONRPCDuplexConnection) Call(ctx context.Context, method string, params any) (JSONRPCResult, error) {
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return JSONRPCResult{}, err
	}
	return c.CallRaw(ctx, method, paramsRaw)
}

func (c *JSONRPCDuplexConnection) CallRaw(ctx context.Context, method string, params json.RawMessage) (JSONRPCResult, error) {
	id := c.lastId.Add(1)

	msgRaw, err := json.Marshal(JSONRPCMessage{
		ID:     id,
		Method: method,
		Params: params,
	})
	if err != nil {
		return JSONRPCResult{}, err
	}

	replyChan := make(chan *JSONRPCMessage, 1)

	c.l.Lock()
	c.pending[id] = replyChan
	c.l.Unlock()

	defer func() {
		c.l.Lock()
		delete(c.pending, id)
		c.l.Unlock()
	}()

	if _, err := c.w.Write(msgRaw); err != nil {
		return JSONRPCResult{}, err
	}

	select {
	case <-ctx.Done():
		return JSONRPCResult{}, ctx.Err()
	case resMsg := <-replyChan:
		return JSONRPCResult{
			ID:     resMsg.ID,
			Result: resMsg.Result,
			Error:  resMsg.Error,
		}, nil
	}
}

func (c *JSONRPCDuplexConnection) Notify(ctx context.Context, method string, params any) error {
	paramsRaw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return c.NotifyRaw(ctx, method, paramsRaw)
}

func (c *JSONRPCDuplexConnection) NotifyRaw(ctx context.Context, method string, params json.RawMessage) error {
	msgRaw, err := json.Marshal(JSONRPCMessage{
		Method: method,
		Params: params,
	})
	if err != nil {
		return err
	}

	if _, err := c.w.Write(msgRaw); err != nil {
		return err
	}

	return nil

}
