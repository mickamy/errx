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

// Attach error details — no transport dependency in your domain/use-case layer
err = errx.New("name is required").
    WithCode(errx.InvalidArgument).
    WithDetails(errx.FieldViolation("name", "must not be empty"))

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

Attach transport-agnostic detail types to errors. The gRPC/Connect interceptors automatically convert them to proto types:

```go
err := errx.New("bad request").
    WithCode(errx.InvalidArgument).
    WithDetails(
        errx.FieldViolation("email", "invalid format"),
        errx.FieldViolation("name", "must not be empty"),
    )

// Collect details from the error chain
details := errx.DetailsOf(err)
```

Available detail types (all in `errx` package):

```go
errx.FieldViolation("email", "invalid format")
errx.BadRequest(errx.BadRequestFieldViolation{Field: "email", Description: "invalid"}, ...)
errx.ResourceInfo("User", "123", "", "not found")
errx.ErrorInfo("QUOTA_EXCEEDED", "example.com", map[string]string{"limit": "100"})
errx.PreconditionFailure(errx.PreconditionViolation{Type: "TOS", Subject: "user", Description: "not accepted"})
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

gRPC integration with code mapping, server interceptors, and infrastructure-level detail helpers.

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
        WithDetails(errx.ResourceInfo("User", req.GetId(), "", "not found"))
}
```

### Infrastructure detail helpers

Helpers for detail types that are typically set at the infrastructure layer:

```go
gerr.QuotaFailure(gerr.NewQuotaViolation("project:abc", "RPM limit exceeded"))
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
