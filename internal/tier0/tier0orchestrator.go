package tier0

import (
	"github.com/rishisinu/Cascade-CI/internal/checks"
	"github.com/rishisinu/Cascade-CI/internal/report"
)

func Run(repo string, base string, binary string) (report.Run, error) {
	run := report.Run{Tier: 0, Status: report.RunPass}

	f, err := checks.GitLeaks(repo, base)

	if err != nil {
		run.Status = report.RunError
		return run, err
	}

	run.Findings = append(run.Findings, f...) // Append takes an arr and what items to add to that arr, so we expand out the obk

	f, err = checks.Semgrep(repo, base)

	if err != nil {
		run.Status = report.RunError
		return run, err
	}

	run.Findings = append(run.Findings, f...)

	if len(run.Findings) > 0 { //If we did finnd errors, fail the run and we will return the findings
		run.Status = report.RunFail
	}

	return run, nil

}
