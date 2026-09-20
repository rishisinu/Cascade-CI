package checks

import (
	"bytes"
	"exec"
)

func useGitLeaks(repo string, base string) {
	//Constructing args so we get correct final output
	args := []string{"git", "--no-banner", "--exit-code", "0",
		"--report-format", "json", "--report-path", "-"}
	if base != "" {
		args = append(args, "--log-opts", base+"..HEAD")
	}
	args = append(args, repo)
	cmd := exec.Command("gitleaks", args...) // THe ... essentially expands the arr so its like you pass in those args individually

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()

}
