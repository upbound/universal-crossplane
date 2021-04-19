package internal

// IdentifierKind - enumeration of supported identifier types
type IdentifierKind string

const (
	// IdentifierKindUserID - know user with user id value
	IdentifierKindUserID IdentifierKind = "userID"
	// IdentifierKindRobotID - known robot's ID
	IdentifierKindRobotID IdentifierKind = "robotID"
	// IdentifierKindAnonymous - anonymous user
	IdentifierKindAnonymous IdentifierKind = "anonymous"
	// OwnerSubjectFormat is the format used in JWT subject claims
	OwnerSubjectFormat string = "%s|%s"
)

// Intentionally copied CrossplaneAccessor struct from https://github.com/upbound/upbound-api/blob/6ec54eb2baf1c9d225ac57aa0359708e85ed491f/pkg/handlers/private_environment.go#L42
// to break wesaas dependency to upbound-api. We don't want this because since upbound-api also depends on wesaas, we
// end up with a circular dependency. This is a quick workaround for this as wesaas has this very small dependency.

// CrossplaneAccessor is the struct holding accessor info in JWT custom claims
type CrossplaneAccessor struct {
	IsOwner        bool           `json:"isOwner"`
	TeamIDs        []string       `json:"teamIds"`
	Identifier     string         `json:"identifier"`
	IdentifierKind IdentifierKind `json:"identifierKind"`
}
