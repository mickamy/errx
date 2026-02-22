# errx

Structured error context for Go. Attach typed codes and slog-native fields to errors without losing `errors.Is`/`errors.As` compatibility.

## Install

```bash
go get github.com/mickamy/errx
```

## Usage

### Create and wrap errors

```go
// New error with structured fields
err := errx.New("user not found", "user_id", 42).WithCode(errx.NotFound)

// Wrap an existing error
err = errx.Wrap(dbErr, "query", q).WithCode(errx.Internal)

// Wrap with a formatted message
err = errx.Wrapf(dbErr, "query %s failed", tableName)
```

### Error codes

Codes are plain strings. Built-in codes map naturally to gRPC/HTTP status codes. Define your own with `const`:

```go
const PaymentRequired errx.Code = "payment_required"

var ErrPaymentRequired = errx.NewSentinel("upgrade needed", PaymentRequired)

errx.CodeOf(ErrPaymentRequired)                  // "payment_required"
errx.IsCode(ErrPaymentRequired, PaymentRequired)  // true
```

### Sentinel errors

```go
var ErrNotFound = errx.NewSentinel("not found", errx.NotFound)

err := errx.Wrap(ErrNotFound, "table", "users")
errors.Is(err, ErrNotFound) // true
errx.CodeOf(err)            // "not_found"
```

### slog integration

`*Error` implements `slog.LogValuer`:

```go
slog.Error("operation failed", "error", err)
// {"level":"ERROR","msg":"operation failed","error":{"msg":"...","code":"not_found","user_id":42}}
```

Or use `SlogAttr` to collect fields from the entire error chain:

```go
slog.Error("failed", errx.SlogAttr(err))
```

### Stack traces

```go
err := errx.New("fail").WithStack()
stack := errx.StackOf(err) // walks the chain
frames := stack.Frames()   // []Frame{Function, File, Line}
```

## License

[MIT](./LICENSE)
