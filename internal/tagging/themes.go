// Package tagging tags reviews with the model: the theme list, the user
// message, the line parser and validator, and the worker that makes one pass
// over the untagged reviews per signal (US-01-003, US-01-004, ADR-0006,
// ADR-0008, phase 3 LLD section 6).
package tagging

// Theme is one entry of the configured theme list.
type Theme struct {
	Code  string
	Label string
}

// Themes is the installation's theme list (REQ-007, REQ-008), edited by a
// developer (Q-012). A change here ships with a new tagging prompt version
// in the same merge request, because the prompt names every code (GenAI
// design 4.3); TestPrompt_CurrentNamesEveryTheme fails otherwise.
var Themes = []Theme{
	{Code: "food", Label: "Food"},
	{Code: "wait_time", Label: "Wait time"},
	{Code: "staff", Label: "Staff"},
	{Code: "cleanliness", Label: "Cleanliness"},
	{Code: "price", Label: "Price"},
}

// UrgentReasons are the three reasons a review is urgent (REQ-010), in the
// order they are stored and returned.
var UrgentReasons = []string{"food_safety", "harassment", "legal_threat"}
