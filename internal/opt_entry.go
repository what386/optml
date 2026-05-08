package internal

import "time"

// OptEntry describes one program installation rooted in /opt/*.
type OptEntry struct {
	// Name is the logical program identifier (for example "node" or "python").
	Name string `json:"name"`

	// Version is the installed program version.
	Version string `json:"version"`

	// RootDir is the absolute install root under /opt
	RootDir string `json:"root_dir"`

	// BinPaths is a list of executables exposed by this installation.
	BinPaths []string `json:"bin_paths"`

	// Managed indicates whether optml manages lifecycle for this entry.
	Managed bool `json:"managed"`

	// InstalledAt is when this entry was installed or first registered.
	InstalledAt time.Time `json:"installed_at"`

	// UpdatedAt is the most recent metadata update time for this entry.
	UpdatedAt time.Time `json:"updated_at"`

	// Checksum is an optional artifact digest used for integrity verification.
	Checksum string `json:"checksum,omitempty"`
}
