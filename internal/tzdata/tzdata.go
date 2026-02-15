package tzdata

// Embed IANA timezone database into the binary so that time.LoadLocation works
// in minimal Docker images (e.g. distroless) that do not ship /usr/share/zoneinfo.
import _ "time/tzdata"
