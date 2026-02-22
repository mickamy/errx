package grpcerr

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/mickamy/errx"
)

// ToGRPCCode maps an errx.Code to a gRPC codes.Code.
// Unknown or user-defined codes map to codes.Unknown.
func ToGRPCCode(c errx.Code) codes.Code {
	if gc, ok := errxToGRPC[c]; ok {
		return gc
	}
	return codes.Unknown
}

// ToErrxCode maps a gRPC codes.Code to an errx.Code.
// codes.OK maps to the zero value ("").
func ToErrxCode(c codes.Code) errx.Code {
	if ec, ok := grpcToErrx[c]; ok {
		return ec
	}
	return errx.Unknown
}

// ToStatus converts an error to a *status.Status.
// If the error carries an errx.Code, it is mapped to a gRPC code.
// The error message is used as the status message.
func ToStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	c := errx.CodeOf(err)
	return status.New(ToGRPCCode(c), err.Error())
}

// FromStatus converts a *status.Status to an *errx.Error.
// Returns nil if the status code is OK.
func FromStatus(st *status.Status) *errx.Error {
	if st.Code() == codes.OK {
		return nil
	}
	return errx.New(st.Message()).WithCode(ToErrxCode(st.Code()))
}

var errxToGRPC = map[errx.Code]codes.Code{
	errx.Canceled:           codes.Canceled,
	errx.Unknown:            codes.Unknown,
	errx.InvalidArgument:    codes.InvalidArgument,
	errx.DeadlineExceeded:   codes.DeadlineExceeded,
	errx.NotFound:           codes.NotFound,
	errx.AlreadyExists:      codes.AlreadyExists,
	errx.PermissionDenied:   codes.PermissionDenied,
	errx.ResourceExhausted:  codes.ResourceExhausted,
	errx.FailedPrecondition: codes.FailedPrecondition,
	errx.Aborted:            codes.Aborted,
	errx.OutOfRange:         codes.OutOfRange,
	errx.Unimplemented:      codes.Unimplemented,
	errx.Internal:           codes.Internal,
	errx.Unavailable:        codes.Unavailable,
	errx.DataLoss:           codes.DataLoss,
	errx.Unauthenticated:    codes.Unauthenticated,
}

var grpcToErrx = map[codes.Code]errx.Code{
	codes.OK:                 "",
	codes.Canceled:           errx.Canceled,
	codes.Unknown:            errx.Unknown,
	codes.InvalidArgument:    errx.InvalidArgument,
	codes.DeadlineExceeded:   errx.DeadlineExceeded,
	codes.NotFound:           errx.NotFound,
	codes.AlreadyExists:      errx.AlreadyExists,
	codes.PermissionDenied:   errx.PermissionDenied,
	codes.ResourceExhausted:  errx.ResourceExhausted,
	codes.FailedPrecondition: errx.FailedPrecondition,
	codes.Aborted:            errx.Aborted,
	codes.OutOfRange:         errx.OutOfRange,
	codes.Unimplemented:      errx.Unimplemented,
	codes.Internal:           errx.Internal,
	codes.Unavailable:        errx.Unavailable,
	codes.DataLoss:           errx.DataLoss,
	codes.Unauthenticated:    errx.Unauthenticated,
}
