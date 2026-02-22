package errx

// BadRequestDetail describes violations in a client request.
type BadRequestDetail struct {
	Violations []BadRequestFieldViolation
}

// BadRequestFieldViolation describes a single field-level violation.
type BadRequestFieldViolation struct {
	Field       string
	Description string
}

// FieldViolation creates a BadRequestDetail with a single field violation.
func FieldViolation(field, description string) *BadRequestDetail {
	return &BadRequestDetail{
		Violations: []BadRequestFieldViolation{{Field: field, Description: description}},
	}
}

// BadRequest creates a BadRequestDetail with the given violations.
func BadRequest(violations ...BadRequestFieldViolation) *BadRequestDetail {
	return &BadRequestDetail{Violations: violations}
}

// PreconditionFailureDetail describes what preconditions were not met.
type PreconditionFailureDetail struct {
	Violations []PreconditionViolation
}

// PreconditionViolation describes a single precondition violation.
type PreconditionViolation struct {
	Type        string
	Subject     string
	Description string
}

// PreconditionFailure creates a PreconditionFailureDetail with the given violations.
func PreconditionFailure(violations ...PreconditionViolation) *PreconditionFailureDetail {
	return &PreconditionFailureDetail{Violations: violations}
}

// ResourceInfoDetail describes the resource that is being accessed.
type ResourceInfoDetail struct {
	ResourceType string
	ResourceName string
	Owner        string
	Description  string
}

// ResourceInfo creates a ResourceInfoDetail.
func ResourceInfo(resourceType, resourceName, owner, description string) *ResourceInfoDetail {
	return &ResourceInfoDetail{
		ResourceType: resourceType,
		ResourceName: resourceName,
		Owner:        owner,
		Description:  description,
	}
}

// ErrorInfoDetail describes the cause of the error with structured details.
type ErrorInfoDetail struct {
	Reason   string
	Domain   string
	Metadata map[string]string
}

// ErrorInfo creates an ErrorInfoDetail.
func ErrorInfo(reason, domain string, metadata map[string]string) *ErrorInfoDetail {
	return &ErrorInfoDetail{
		Reason:   reason,
		Domain:   domain,
		Metadata: metadata,
	}
}
