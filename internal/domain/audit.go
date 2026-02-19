package domain

// AuditStep represents one discrete step in a calculation audit trail.
// Each step documents a single formula application with its inputs, calculation string,
// and numeric result — providing full transparency for regulatory review and debugging.
type AuditStep struct {
	StepNumber  int                    `json:"stepNumber"`
	StepName    string                 `json:"stepName"`
	Description string                 `json:"description"`
	Formula     string                 `json:"formula"`
	Inputs      map[string]interface{} `json:"inputs"`
	Calculation string                 `json:"calculation"`
	// Result is stored as float64 for JSON readability; built from decimal.Decimal.InexactFloat64()
	Result float64 `json:"result"`
	Notes  string  `json:"notes,omitempty"`
}

// CalculationAuditTrail documents a complete calculation with a step-by-step breakdown.
// It is attached to calculation results when the caller requests audit detail (e.g. via
// the web API ?audit=true query parameter or test assertions).
type CalculationAuditTrail struct {
	// CalculationType identifies what was calculated, e.g. "FERS Pension"
	CalculationType string `json:"calculationType"`

	// InputSummary is a human-readable one-line summary of the key inputs
	InputSummary string `json:"inputSummary"`

	// Steps contains each calculation step in order
	Steps []AuditStep `json:"steps"`

	// FinalResult is the primary output of the calculation (e.g. annual reduced pension)
	FinalResult float64 `json:"finalResult"`

	// Warnings flags important conditions that affect the result (e.g. permanent reductions)
	Warnings []string `json:"warnings,omitempty"`

	// OPMReferences cites the specific OPM/IRS/USC provisions that govern the calculation
	OPMReferences []string `json:"opmReferences,omitempty"`
}
