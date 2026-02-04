package lineage

import "testing"

func TestIsRescueHostname(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		"rescue":     true,
		"rescue-123": true,
		"RESCUE-42":  true,
		"builder":    false,
		"root":       false,
	}
	for hostname, expected := range cases {
		if isRescueHostname(hostname) != expected {
			t.Fatalf("hostname %q expected %v", hostname, expected)
		}
	}
}

func TestIsRescueRootFilesystem(t *testing.T) {
	t.Parallel()

	rescueOutput := `Filesystem     Type 1K-blocks    Used Available Use% Mounted on
tmpfs          tmpfs   3280256       0   3280256   0% /`
	normalOutput := `Filesystem     Type 1K-blocks    Used Available Use% Mounted on
/dev/sda2      ext4   61888508 8869232  49858012  16% /`

	if !isRescueRootFilesystem(rescueOutput) {
		t.Fatalf("expected rescue filesystem for tmpfs")
	}
	if isRescueRootFilesystem(normalOutput) {
		t.Fatalf("did not expect rescue filesystem for ext4")
	}
}
