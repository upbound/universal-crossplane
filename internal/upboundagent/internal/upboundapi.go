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

// CrossplaneAccessor is the struct holding accessor info in JWT custom claims
type CrossplaneAccessor struct {
	IsOwner        bool           `json:"isOwner"`
	TeamIDs        []string       `json:"teamIds"`
	Identifier     string         `json:"identifier"`
	IdentifierKind IdentifierKind `json:"identifierKind"`
}
