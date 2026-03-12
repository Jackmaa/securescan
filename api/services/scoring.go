package services

import "securescan/models"

// ComputeScore calculates a security score (0-100) and letter grade from findings.
//
// The algorithm applies weighted penalties per severity level:
//   critical=10, high=5, medium=2, low=0.5
//
// Score starts at 100 and is reduced by the total penalty, clamped to [0, 100].
// This intentionally penalizes critical findings harshly — a repo with even a few
// critical issues will score poorly, reflecting real-world risk.
//
// Grade thresholds:
//   A(≥90) | B(≥75) | C(≥55) | D(≥35) | F(<35)
func ComputeScore(findings []models.Finding) (int, string) {
	var penalty float64
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			penalty += 10
		case "high":
			penalty += 5
		case "medium":
			penalty += 2
		case "low":
			penalty += 0.5
		}
	}

	score := int(100 - penalty)
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	grade := "F"
	switch {
	case score >= 90:
		grade = "A"
	case score >= 75:
		grade = "B"
	case score >= 55:
		grade = "C"
	case score >= 35:
		grade = "D"
	}

	return score, grade
}
