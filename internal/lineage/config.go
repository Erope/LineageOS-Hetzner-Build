package lineage

type Config struct {
	HetznerToken        string
	ServerType          string
	ServerLocation      string
	ServerImage         string
	ServerName          string
	ServerUserDataPath  string
	InstanceSSHUser     string
	InstanceSSHKeyPath  string
	KnownHostsPath      string
	BuildRepoURL        string
	BuildRepoRef        string
	ComposeFile         string
	WorkingDir          string
	ReleaseRepoOwner    string
	ReleaseRepoName     string
	ReleaseTag          string
	ReleaseName         string
	ReleaseNotes        string
	GitHubToken         string
	ArtifactDir         string
	ArtifactPattern     string
	LocalArtifactDir    string
	SSHPort             int
	BuildTimeoutMinutes int
}

func (c Config) HasReleaseTarget() bool {
	return c.ReleaseRepoOwner != "" && c.ReleaseRepoName != "" && c.ReleaseTag != ""
}

func (c Config) HasSSHConfig() bool {
	return c.InstanceSSHUser != "" && c.InstanceSSHKeyPath != "" && c.KnownHostsPath != ""
}
