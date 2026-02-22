package cerr

import (
	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/mickamy/errx"
)

// ToConnectCode maps an errx.Code to a connect.Code.
// Unknown or user-defined codes map to connect.CodeUnknown.
func ToConnectCode(c errx.Code) connect.Code {
	if cc, ok := errxToConnect[c]; ok {
		return cc
	}
	return connect.CodeUnknown
}

// ToErrxCode maps a connect.Code to an errx.Code.
// Zero value (no error) maps to the zero value ("").
func ToErrxCode(c connect.Code) errx.Code {
	if ec, ok := connectToErrx[c]; ok {
		return ec
	}
	return errx.Unknown
}

// ToConnectError converts an error to a *connect.Error.
// If the error carries an errx.Code, it is mapped to a Connect code.
// Any detail objects (proto.Message) attached via errx.WithDetails are
// included as Connect error details. Non-proto.Message details are ignored.
func ToConnectError(err error) *connect.Error {
	if err == nil {
		return nil
	}
	c := errx.CodeOf(err)
	ce := connect.NewError(ToConnectCode(c), err)

	for _, d := range errx.DetailsOf(err) {
		pm, ok := d.(proto.Message)
		if !ok {
			continue
		}
		detail, detailErr := connect.NewErrorDetail(pm)
		if detailErr != nil {
			continue
		}
		ce.AddDetail(detail)
	}
	return ce
}

// FromConnectError converts a *connect.Error to an *errx.Error.
// Returns nil if err is nil.
// Any Connect error details are restored via errx.WithDetails.
func FromConnectError(err *connect.Error) *errx.Error {
	if err == nil {
		return nil
	}
	ex := errx.New(err.Message()).WithCode(ToErrxCode(err.Code()))
	var details []any
	for _, d := range err.Details() {
		v, valErr := d.Value()
		if valErr != nil {
			continue
		}
		details = append(details, v)
	}
	if len(details) > 0 {
		ex = ex.WithDetails(details...)
	}
	return ex
}

var errxToConnect = map[errx.Code]connect.Code{
	errx.Canceled:           connect.CodeCanceled,
	errx.Unknown:            connect.CodeUnknown,
	errx.InvalidArgument:    connect.CodeInvalidArgument,
	errx.DeadlineExceeded:   connect.CodeDeadlineExceeded,
	errx.NotFound:           connect.CodeNotFound,
	errx.AlreadyExists:      connect.CodeAlreadyExists,
	errx.PermissionDenied:   connect.CodePermissionDenied,
	errx.ResourceExhausted:  connect.CodeResourceExhausted,
	errx.FailedPrecondition: connect.CodeFailedPrecondition,
	errx.Aborted:            connect.CodeAborted,
	errx.OutOfRange:         connect.CodeOutOfRange,
	errx.Unimplemented:      connect.CodeUnimplemented,
	errx.Internal:           connect.CodeInternal,
	errx.Unavailable:        connect.CodeUnavailable,
	errx.DataLoss:           connect.CodeDataLoss,
	errx.Unauthenticated:    connect.CodeUnauthenticated,
}

var connectToErrx = map[connect.Code]errx.Code{
	0:                              "",
	connect.CodeCanceled:           errx.Canceled,
	connect.CodeUnknown:            errx.Unknown,
	connect.CodeInvalidArgument:    errx.InvalidArgument,
	connect.CodeDeadlineExceeded:   errx.DeadlineExceeded,
	connect.CodeNotFound:           errx.NotFound,
	connect.CodeAlreadyExists:      errx.AlreadyExists,
	connect.CodePermissionDenied:   errx.PermissionDenied,
	connect.CodeResourceExhausted:  errx.ResourceExhausted,
	connect.CodeFailedPrecondition: errx.FailedPrecondition,
	connect.CodeAborted:            errx.Aborted,
	connect.CodeOutOfRange:         errx.OutOfRange,
	connect.CodeUnimplemented:      errx.Unimplemented,
	connect.CodeInternal:           errx.Internal,
	connect.CodeUnavailable:        errx.Unavailable,
	connect.CodeDataLoss:           errx.DataLoss,
	connect.CodeUnauthenticated:    errx.Unauthenticated,
}
