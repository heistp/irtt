package irtt

// Version is the IRTT version number (replaced during build).
var Version = "0.9"

// BuildDate is the build date (replaced during build).
var BuildDate = "unavailable"

// ProtocolVersion is the protocol version number, which must match between client
// and server.
var ProtocolVersion = 1

// JSONFormatVersion is the JSON format number.
var JSONFormatVersion = 1

// VersionInfo stores the version information.
type VersionInfo struct {
	IRTT       string `json:"irtt"`
	BuildDate  string `json:"build_date"`
	Protocol   int    `json:"protocol"`
	JSONFormat int    `json:"json_format"`
}

// NewVersionInfo returns a new VersionInfo.
func NewVersionInfo() *VersionInfo {
	return &VersionInfo{
		IRTT:       Version,
		BuildDate:  BuildDate,
		Protocol:   ProtocolVersion,
		JSONFormat: JSONFormatVersion,
	}
}
