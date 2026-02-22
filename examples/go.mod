module github.com/mickamy/errx/examples

go 1.25.0

require (
	connectrpc.com/connect v1.19.1
	github.com/mickamy/errx v0.0.1
	github.com/mickamy/errx/cerr v0.0.0
	github.com/mickamy/errx/gerr v0.0.0
	github.com/mickamy/errx/herr v0.0.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260217215200-42d3e9bedb6d
	google.golang.org/grpc v1.79.1
	google.golang.org/grpc/examples v0.0.0-20260220101807-19e41284feb7
)

require (
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/mickamy/errx => ../
	github.com/mickamy/errx/cerr => ../cerr
	github.com/mickamy/errx/gerr => ../gerr
	github.com/mickamy/errx/herr => ../herr
)
