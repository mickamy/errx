package herr

import (
	"encoding/json"
	"net/http"

	"github.com/mickamy/errx"
)

// RegisterCode registers a custom mapping between an errx.Code and an HTTP status code.
// Both forward (errx → HTTP) and reverse (HTTP → errx) mappings are registered.
// Must be called at program initialization (e.g. in init()), before serving requests.
func RegisterCode(c errx.Code, status int) {
	errxToHTTP[c] = status
	httpToErrx[status] = c
}

// ToHTTPStatus maps an errx.Code to an HTTP status code.
// Unknown or user-defined codes map to 500.
func ToHTTPStatus(c errx.Code) int {
	if s, ok := errxToHTTP[c]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// ToErrxCode maps an HTTP status code to an errx.Code.
// Unmapped status codes return errx.Unknown.
func ToErrxCode(status int) errx.Code {
	if c, ok := httpToErrx[status]; ok {
		return c
	}
	return errx.Unknown
}

// ProblemDetail is an RFC 9457 Problem Details response.
// Standard members (type, title, status, detail, instance) follow the spec.
// Extension members (code, errors, localized_message) carry errx-specific data.
type ProblemDetail struct {
	// RFC 9457 standard members.
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance,omitempty"`

	// Extension members.
	Code             string           `json:"code,omitempty"`
	Errors           []map[string]any `json:"errors,omitempty"`
	LocalizedMessage *LocalizedMsg    `json:"localized_message,omitempty"`
}

// LocalizedMsg holds a locale-specific error message.
type LocalizedMsg struct {
	Locale  string `json:"locale"`
	Message string `json:"message"`
}

// ProblemDetailOption configures a [ProblemDetail] built by [ToProblemDetail].
type ProblemDetailOption func(*ProblemDetail)

// WithInstance sets the instance URI on the problem detail.
func WithInstance(instance string) ProblemDetailOption {
	return func(p *ProblemDetail) {
		p.Instance = instance
	}
}

// WithType sets the type URI on the problem detail.
// Defaults to "about:blank" if not set.
func WithType(typeURI string) ProblemDetailOption {
	return func(p *ProblemDetail) {
		p.Type = typeURI
	}
}

// ToProblemDetail converts an error to an RFC 9457 [ProblemDetail].
// Returns nil if err is nil.
func ToProblemDetail(err error, opts ...ProblemDetailOption) *ProblemDetail {
	if err == nil {
		return nil
	}
	c := errx.CodeOf(err)
	status := ToHTTPStatus(c)

	code := string(c)
	if code == "" {
		code = string(errx.Unknown)
	}

	title := http.StatusText(status)
	if title == "" {
		title = code
	}

	p := &ProblemDetail{
		Type:   "about:blank",
		Title:  title,
		Status: status,
		Detail: err.Error(),
		Code:   code,
	}

	for _, d := range errx.DetailsOf(err) {
		if m := toDetailJSON(d); m != nil {
			p.Errors = append(p.Errors, m)
		}
	}

	for _, o := range opts {
		o(p)
	}

	return p
}

// FromProblemDetail converts an RFC 9457 [ProblemDetail] back to an [*errx.Error].
// Returns nil if p is nil.
func FromProblemDetail(p *ProblemDetail) *errx.Error {
	if p == nil {
		return nil
	}
	code := errx.Code(p.Code)
	if code == "" {
		code = ToErrxCode(p.Status)
	}
	return errx.New(p.Detail).WithCode(code)
}

// WriteError writes an RFC 9457 JSON error response to w.
// Does nothing if err is nil.
func WriteError(w http.ResponseWriter, err error, opts ...ProblemDetailOption) {
	if err == nil {
		return
	}
	writeProblemDetail(w, ToProblemDetail(err, opts...))
}

func writeProblemDetail(w http.ResponseWriter, p *ProblemDetail) {
	b, marshalErr := json.Marshal(p)
	if marshalErr != nil {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"type":"about:blank","title":"Internal Server Error","status":500}` + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.Status)
	_, _ = w.Write(b)
	_, _ = w.Write([]byte("\n"))
}

func toDetailJSON(d any) map[string]any {
	switch v := d.(type) {
	case *errx.BadRequestDetail:
		violations := make([]map[string]any, len(v.Violations))
		for i, fv := range v.Violations {
			violations[i] = map[string]any{
				"field":       fv.Field,
				"description": fv.Description,
			}
		}
		return map[string]any{
			"type":       "BadRequest",
			"violations": violations,
		}
	case *errx.PreconditionFailureDetail:
		violations := make([]map[string]any, len(v.Violations))
		for i, pv := range v.Violations {
			violations[i] = map[string]any{
				"type":        pv.Type,
				"subject":     pv.Subject,
				"description": pv.Description,
			}
		}
		return map[string]any{
			"type":       "PreconditionFailure",
			"violations": violations,
		}
	case *errx.ResourceInfoDetail:
		return map[string]any{
			"type":          "ResourceInfo",
			"resource_type": v.ResourceType,
			"resource_name": v.ResourceName,
			"owner":         v.Owner,
			"description":   v.Description,
		}
	case *errx.ErrorInfoDetail:
		return map[string]any{
			"type":     "ErrorInfo",
			"reason":   v.Reason,
			"domain":   v.Domain,
			"metadata": v.Metadata,
		}
	default:
		return nil
	}
}

var errxToHTTP = map[errx.Code]int{
	errx.InvalidArgument:    http.StatusBadRequest,
	errx.OutOfRange:         http.StatusBadRequest,
	errx.Unauthenticated:    http.StatusUnauthorized,
	errx.PermissionDenied:   http.StatusForbidden,
	errx.NotFound:           http.StatusNotFound,
	errx.AlreadyExists:      http.StatusConflict,
	errx.Aborted:            http.StatusConflict,
	errx.FailedPrecondition: http.StatusPreconditionFailed,
	errx.ResourceExhausted:  http.StatusTooManyRequests,
	errx.Canceled:           499,
	errx.Internal:           http.StatusInternalServerError,
	errx.Unknown:            http.StatusInternalServerError,
	errx.DataLoss:           http.StatusInternalServerError,
	errx.Unimplemented:      http.StatusNotImplemented,
	errx.Unavailable:        http.StatusServiceUnavailable,
	errx.DeadlineExceeded:   http.StatusGatewayTimeout,
}

var httpToErrx = map[int]errx.Code{
	http.StatusBadRequest:          errx.InvalidArgument,
	http.StatusUnauthorized:        errx.Unauthenticated,
	http.StatusForbidden:           errx.PermissionDenied,
	http.StatusNotFound:            errx.NotFound,
	http.StatusConflict:            errx.AlreadyExists,
	http.StatusPreconditionFailed:  errx.FailedPrecondition,
	http.StatusTooManyRequests:     errx.ResourceExhausted,
	499:                            errx.Canceled,
	http.StatusInternalServerError: errx.Internal,
	http.StatusNotImplemented:      errx.Unimplemented,
	http.StatusServiceUnavailable:  errx.Unavailable,
	http.StatusGatewayTimeout:      errx.DeadlineExceeded,
}
