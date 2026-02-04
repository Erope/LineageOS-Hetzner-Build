package lineage

type Config struct {
	HetznerToken        string
	ServerType          string
	ServerLocation      string
	ServerImage         string
	ServerName          string
	ServerUserDataPath  string
	BuildRepoURL        string
	BuildRepoRef        string
	BuildRepoToken      string
	BuildRepoOwner      string
	BuildRepoName       string
	BuildRepoHost       string
	BuildRepoSHA        string
	ComposeFile         string
	WorkingDir          string
	GitHubToken         string
	ArtifactDir         string
	ArtifactPattern     string
	LocalArtifactDir    string
	SSHPort             int
	BuildTimeoutMinutes int
}
