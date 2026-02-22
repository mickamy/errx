# errx

Structured errors for Go with first-class gRPC / Connect support.

- **Structured context** — attach typed codes and slog-native fields while keeping `errors.Is`/`errors.As` compatibility
- **gRPC / Connect error details** — carry [google.rpc.error_details](https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto) (FieldViolation, ResourceInfo, etc.) on any error and let the interceptor convert them automatically
- **Localization** — implement `errx.Localizable` on your domain errors; the interceptor auto-appends `LocalizedMessage` based on `Accept-Language`

## Install

```bash
# core
go get github.com/mickamy/errx

# gRPC integration
go get github.com/mickamy/errx/gerr

# Connect integration
go get github.com/mickamy/errx/cerr
```

## Quick start

```go
// Create an error with a code and structured fields
err := errx.New("user not found", "user_id", 42).WithCode(errx.NotFound)

// Attach gRPC error details
err = errx.New("name is required").
    WithCode(errx.InvalidArgument).
    WithDetails(gerr.FieldViolation("name", "must not be empty"))

// The gRPC/Connect interceptor converts errx errors automatically —
// handlers just return errors, no manual status construction needed.
```

## errx (core)

### Create and wrap errors

```go
err := errx.New("user not found", "user_id", 42).WithCode(errx.NotFound)

err = errx.Wrap(dbErr, "query", q).WithCode(errx.Internal)

err = errx.Wrapf(dbErr, "query %s failed", tableName)
```

### Error codes

Codes are plain strings. Built-in codes map to gRPC/Connect status codes. Define your own:

```go
const PaymentRequired errx.Code = "payment_required"

var ErrPaymentRequired = errx.NewSentinel("upgrade needed", PaymentRequired)

errx.CodeOf(ErrPaymentRequired)                  // "payment_required"
errx.IsCode(ErrPaymentRequired, PaymentRequired) // true
```

### Sentinel errors

```go
var ErrNotFound = errx.NewSentinel("not found", errx.NotFound)

err := errx.Wrap(ErrNotFound, "table", "users")
errors.Is(err, ErrNotFound) // true
errx.CodeOf(err)            // "not_found"
```

### Error details

Attach arbitrary detail objects (typically `proto.Message`) to errors. The gRPC/Connect interceptors pick them up automatically:

```go
err := errx.New("bad request").
    WithCode(errx.InvalidArgument).
    WithDetails(
        gerr.FieldViolation("email", "invalid format"),
        gerr.FieldViolation("name", "must not be empty"),
    )

// Collect details from the error chain
details := errx.DetailsOf(err)
```

### Localization

Implement `errx.Localizable` on your domain errors:

```go
type ValidationError struct {
    Field    string
    Messages map[string]string // locale -> message
}

func (e *ValidationError) Error() string          { return e.Field + " is invalid" }
func (e *ValidationError) Localize(locale string) string { return e.Messages[locale] }
```

The interceptor automatically appends a `LocalizedMessage` detail based on the request's `Accept-Language` header. No extra code in your handlers.

### slog integration

`*Error` implements `slog.LogValuer`:

```go
slog.Error("operation failed", "error", err)
// {"level":"ERROR","msg":"operation failed","error":{"msg":"...","code":"not_found","user_id":42}}
```

Collect fields from the entire error chain:

```go
slog.Error("failed", errx.SlogAttr(err))
```

### Stack traces

```go
err := errx.New("fail").WithStack()
stack := errx.StackOf(err)
frames := stack.Frames() // []Frame{Function, File, Line}
```

## gerr (gRPC)

gRPC integration with code mapping, error detail helpers, and server interceptors.

### Interceptors

```go
srv := grpc.NewServer(
    grpc.UnaryInterceptor(gerr.UnaryServerInterceptor()),
    grpc.StreamInterceptor(gerr.StreamServerInterceptor()),
)
```

Handlers just return `errx` errors — the interceptor converts them to gRPC status errors with details:

```go
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    // Just return an errx error. The interceptor handles the rest.
    return nil, errx.Wrap(ErrUserNotFound).
        WithDetails(gerr.ResourceInfo("User", req.GetId(), "", "not found"))
}
```

### Error detail helpers

Constructors for all [google.rpc.error_details](https://github.com/googleapis/googleapis/blob/master/google/rpc/error_details.proto) types:

```go
gerr.FieldViolation("email", "invalid format")
gerr.BadRequest(gerr.NewFieldViolation("email", "invalid"), gerr.NewFieldViolation("name", "required"))
gerr.ResourceInfo("User", "123", "", "not found")
gerr.ErrorInfo("QUOTA_EXCEEDED", "example.com", map[string]string{"limit": "100"})
gerr.PreconditionFailure(gerr.NewPreconditionViolation("TOS", "user", "Terms not accepted"))
gerr.RetryInfo(5 * time.Second)
gerr.DebugInfo([]string{"main.go:42"}, "nil pointer")
gerr.LocalizedMessage("ja", "名前は必須です")
```

### Round-trip conversion

```go
st := gerr.ToStatus(err)        // errx → gRPC status
ex := gerr.FromStatus(st)       // gRPC status → errx (with details restored)
```

## cerr (Connect)

Connect RPC integration with the same code mapping and interceptor pattern.

### Interceptor

```go
interceptor := cerr.NewInterceptor()
_, handler := foov1connect.NewFooServiceHandler(svc,
    connect.WithInterceptors(interceptor),
)
```

### Localization with custom locale extraction

```go
interceptor := cerr.NewInterceptor(
    cerr.WithLocaleFunc(func(h http.Header) string {
        // Custom logic: cookie, query param, etc.
        return h.Get("X-Locale")
    }),
)
```

### Conversion functions

```go
ce := cerr.ToConnectError(err)       // errx → *connect.Error (with details)
ex := cerr.FromConnectError(ce)      // *connect.Error → *errx.Error (with details)
```

## License

[MIT](./LICENSE)
