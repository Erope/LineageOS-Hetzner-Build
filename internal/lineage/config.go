package lineage

type Config struct {
	HetznerToken        string
	ServerType          string
	ServerLocation      string
	ServerImage         string
	ServerName          string
	ServerUserDataPath  string
	BuildSourceDir      string
	ComposeFile         string
	WorkingDir          string
	ArtifactDir         string
	ArtifactPattern     string
	LocalArtifactDir    string
	SSHPort             int
	BuildTimeoutMinutes int
}
