package report

type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type SourceTool string

const (
	ToolGitleaks   SourceTool = "gitleaks"
	ToolSemgrep    SourceTool = "semgrep"
	ToolElfHygiene SourceTool = "elf-hygiene"
)

type Location struct {
	File string `json:"file"`
	Line int    `json:"line,omitempty"`
}

type Finding struct {
	ID         string     `json:"id"`
	Tier       string     `json:"tier"`
	SourceTool SourceTool `json:"source_tool"`
	Severity   Severity   `json:"severity"`
	Category   string     `json:"category"`
}

type RunStatus string

const (
	RunPass  RunStatus = "pass"
	RunFail  RunStatus = "fail"
	RunError RunStatus = "error"
)

type Run struct {
	CommitSHA    string            `json:"commitSHA"`
	Tier         int               `json:"tier"`
	Status       RunStatus         `json:"status"`
	ToolVersions map[string]string `json:"tool_versions"`
	Findings     []Finding         `json:"findings"`
}
