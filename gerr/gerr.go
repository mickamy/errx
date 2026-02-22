package gerr

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/protoadapt"

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
// Any detail objects (proto.Message) attached via errx.WithDetails are
// included as gRPC status details. Non-proto.Message details are ignored.
func ToStatus(err error) *status.Status {
	if err == nil {
		return status.New(codes.OK, "")
	}
	c := errx.CodeOf(err)
	st := status.New(ToGRPCCode(c), err.Error())

	var protoDetails []protoadapt.MessageV1
	for _, d := range errx.DetailsOf(err) {
		if pm := toProtoDetail(d); pm != nil {
			protoDetails = append(protoDetails, pm)
		}
	}
	if len(protoDetails) > 0 {
		if withDetails, detailErr := st.WithDetails(protoDetails...); detailErr == nil {
			st = withDetails
		}
	}
	return st
}

// FromStatus converts a *status.Status to an *errx.Error.
// Returns nil if the status code is OK.
// Any gRPC status details are restored via errx.WithDetails.
func FromStatus(st *status.Status) *errx.Error {
	if st.Code() == codes.OK {
		return nil
	}
	err := errx.New(st.Message()).WithCode(ToErrxCode(st.Code()))
	if details := st.Details(); len(details) > 0 {
		err = err.WithDetails(details...)
	}
	return err
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

// toProtoDetail converts an errx detail type to a proto.Message.
// If the detail is already a proto.Message, it is returned as-is.
// Returns nil for unrecognized types.
func toProtoDetail(d any) protoadapt.MessageV1 {
	switch v := d.(type) {
	case *errx.BadRequestDetail:
		violations := make([]*errdetails.BadRequest_FieldViolation, len(v.Violations))
		for i, fv := range v.Violations {
			violations[i] = &errdetails.BadRequest_FieldViolation{
				Field:       fv.Field,
				Description: fv.Description,
			}
		}
		return &errdetails.BadRequest{FieldViolations: violations}
	case *errx.PreconditionFailureDetail:
		violations := make([]*errdetails.PreconditionFailure_Violation, len(v.Violations))
		for i, pv := range v.Violations {
			violations[i] = &errdetails.PreconditionFailure_Violation{
				Type:        pv.Type,
				Subject:     pv.Subject,
				Description: pv.Description,
			}
		}
		return &errdetails.PreconditionFailure{Violations: violations}
	case *errx.ResourceInfoDetail:
		return &errdetails.ResourceInfo{
			ResourceType: v.ResourceType,
			ResourceName: v.ResourceName,
			Owner:        v.Owner,
			Description:  v.Description,
		}
	case *errx.ErrorInfoDetail:
		return &errdetails.ErrorInfo{
			Reason:   v.Reason,
			Domain:   v.Domain,
			Metadata: v.Metadata,
		}
	default:
		if pm, ok := d.(protoadapt.MessageV1); ok {
			return pm
		}
		return nil
	}
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
