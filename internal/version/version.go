package version

// Version information
const (
	// Version is the current version of TenyoJubaku
	// Format: VX.Y where X is major version (feature updates), Y is minor version (bug fixes)
	Version = "V4.4"

	// Build information (can be set during build time using ldflags)
	BuildDate = "2026-01-06"
)

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns version with build date
func GetFullVersion() string {
	return Version + " (built on " + BuildDate + ")"
}
