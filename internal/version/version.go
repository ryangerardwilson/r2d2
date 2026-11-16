package version

// Version is the r2d2 release. install.sh stamps the commit beside it.
var (
	Version    = "0.1.0"
	Commit     = ""
	SourceHash = ""
	BuildTime  = ""
)

// String is the version stamped into the r2d2 binary.
func String() string {
	if Commit == "" || Commit == "unknown" {
		return Version
	}
	return Version + "+" + Commit
}
