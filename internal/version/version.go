package version

// Version information
const (
	// Version is the current version of TenyoJubaku
	// Format: VX.Y where X is major version (feature updates), Y is minor version (bug fixes)
	Version = "V3.0"

	// Build information (can be set during build time using ldflags)
	BuildDate = "2025-12-25"
)

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns version with build date
func GetFullVersion() string {
	return Version + " (built on " + BuildDate + ")"
}
