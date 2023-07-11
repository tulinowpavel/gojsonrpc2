# Golang JSONRPC 2.0 Client/Server Duplex protocol implementation

## Links
- [jsonrpc 2.0 specification](https://www.jsonrpc.org/specification)

## Usage examples

### Register methods on dispatcher

```go
dispatcher := gojsonrpc2.NewJSONRPCDispatcher()

dispatcher.Method("echo").
    SetDocs("Return params as result").
    SetHandlerFunc(func(ctx context.Context, r *gojsonrpc2.JSONRPCRequest) {
        _ = r.RespondRaw(ctx, r.Msg.Params)
    }).Register()

dispatcher.Method("sum").
    SetDocs("Return sum of x and y").
    DefineError(-1000, "some banana error").
    SetHandler(gojsonrpc2.NewTypedHandler(
        func(ctx context.Context, params struct {
            X int `json:"x"`
            Y int `json:"y"`
        }) (int, error) {
            return params.X + params.Y, nil
        },
    )).Register()
```

### Serve over net/http

```go
http.ListenAndServe(
    "0.0.0.0:8080",
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // TODO: authenticate here and add principal to context as value

        rawBody, err := io.ReadAll(r.Body)
        if err != nil {
            log.Printf("cannot read http body: %v", err)
            return
        }
        r.Body.Close()

        w.WriteHeader(http.StatusOK)
        w.Header().Add("Content-Type", "application/json")

        if err := dispatcher.ServeRaw(ctx, w, rawBody); err != nil {
            log.Printf("jsonrpc call over http request serve error: %v", err)
        }
    }),
)
```

### Serve over websocket

```go
http.ListenAndServe(
    "0.0.0.0:8080",
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()

        // TODO: authenticate request and add principal to context as value

        conn, _, _, err := ws.UpgradeHTTP(r, w)
        if err != nil {
            log.Printf("upgrade http error: %v", err)
            return
        }

        wsw := gojsonrpc2.WriterFunc(func(p []byte) (int, error) {
            if err := wsutil.WriteServerText(conn, p); err != nil {
                return 0, err
            }
            return len(p), nil
        })

        for {
            msg, err := wsutil.ReadClientText(conn)
            if err != nil {
                if errors.As(err, &wsutil.ClosedError{}) {
                    log.Println("ws connection closed")
                    return
                }

                log.Printf("read client message error: %v", err)
                continue
            }

            if err := dispatcher.ServeRaw(ctx, wsw, msg); err != nil {
                log.Printf("error during serve jsonrpc request occurred: %v", err)
            }
        }

    }),
)
```

### Serve over TCP

TODO