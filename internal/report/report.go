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
	Tier       int        `json:"tier"`
	SourceTool SourceTool `json:"source_tool"`
	Severity   Severity   `json:"severity"`
	Category   string     `json:"category"`
	RawMessage any        `json:"raw_message"`
	Location   Location   `json:"location"`
	//Tier 1 Findings
	Verdict     string      `json:"verdict,omitempty"`
	Confidence  string      `json:"confidence,omitempty"`
	Trace       []TraceStep `json:"trace,omitempty"`
	Remediation string      `json:"remediation,omitempty"`
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

const (
	ToolGitLeaks SourceTool = "gitleaks"
	ToolSemgrep  SourceTool = "semgrep"
	ToolElf      SourceTool = "elf"
	ToolGosec    SourceTool = "gosec"
)

type TraceStep struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Note string `json:"note"`
}
