package ollama

import (
	"regexp"
	"strings"
)

var (
	dateRe = regexp.MustCompile(
		`(?i)\b(?:` +
			`\d{1,2}[/-]\d{1,2}(?:[/-]\d{2,4})?` +
			`|` +
			`(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\s+\d{1,2}` +
			`)\b`,
	)

	// Require either a decimal amount or a currency symbol.
	// This prevents plain integers such as "2026" or "31"
	// from being treated as transaction amounts.
	amountRe = regexp.MustCompile(
		`(?i)(?:` +
			`[+-]?\s*\$?\d[\d,]*\.\d{2}` +
			`|` +
			`\(\s*\$?\d[\d,]*\.\d{2}\s*\)` +
			`|` +
			`[+-]?\s*\$\d[\d,]*(?:\.\d{2})?` +
			`)`,
	)

	whitespaceRe = regexp.MustCompile(`\s+`)
)

func NormalizeStatementText(input string) string {
	lines := cleanLines(input)

	regions := findTransactionRegions(lines)

	var result []string

	for _, region := range regions {
		result = append(result, region...)
	}

	return strings.Join(result, "\n")
}

func cleanLines(input string) []string {
	raw := strings.Split(input, "\n")

	lines := make([]string, 0, len(raw))

	for _, line := range raw {
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		line = whitespaceRe.ReplaceAllString(line, " ")

		lines = append(lines, line)
	}

	return lines
}

func findTransactionRegions(lines []string) [][]string {
	var regions [][]string

	start := -1
	lastSignal := -1
	signalCount := 0

	for i, line := range lines {
		if isTransactionSignal(line) {
			if start == -1 {
				start = max(0, i-2)
			}

			lastSignal = i
			signalCount++
			continue
		}

		if start == -1 {
			continue
		}

		// A transaction can have several lines between
		// its date and amount/description.
		//
		// Once we've gone more than 4 lines without seeing
		// another transaction signal, the region is probably over.
		if i-lastSignal > 4 {
			if signalCount >= 2 {
				region := extractRegion(lines, start, lastSignal+2)

				if len(region) > 0 {
					regions = append(regions, region)
				}
			}

			start = -1
			lastSignal = -1
			signalCount = 0
		}
	}

	// Handle a transaction region reaching EOF.
	if start != -1 && signalCount >= 2 {
		end := min(len(lines), lastSignal+3)
		region := extractRegion(lines, start, end)

		if len(region) > 0 {
			regions = append(regions, region)
		}
	}

	return regions
}

func isTransactionSignal(line string) bool {
	return dateRe.MatchString(line) || amountRe.MatchString(line)
}

func extractRegion(lines []string, start, end int) []string {
	end = min(end, len(lines))

	var result []string

	for i := start; i < end; i++ {
		line := lines[i]

		if isNoise(line) {
			continue
		}

		if isTransactionSignal(line) || isLikelyDescription(line) {
			result = appendUnique(result, line)
		}
	}

	return result
}

func isLikelyDescription(line string) bool {
	line = strings.TrimSpace(line)

	if line == "" {
		return false
	}

	// Legal/disclaimer paragraphs are generally much longer.
	if len(line) > 160 {
		return false
	}

	// A pure number is not a description.
	if amountRe.MatchString(line) && len(strings.Fields(line)) == 1 {
		return false
	}

	// Don't keep obvious document-level text.
	lower := strings.ToLower(line)

	noise := []string{
		"page ",
		"billing cycle",
		"statement period",
		"account ending",
		"account number",
		"customer service",
		"customer support",
		"important information",
		"terms and conditions",
		"annual percentage",
		"interest rate",
		"interest charge",
		"minimum payment",
		"payment due",
		"previous balance",
		"new balance",
		"available credit",
		"credit limit",
		"total transactions",
		"total interest",
		"year-to-date",
		"amount enclosed",
		"po box",
		"copyright",
		"how do i",
		"how can i",
		"your rights",
	}

	for _, phrase := range noise {
		if strings.Contains(lower, phrase) {
			return false
		}
	}

	return true
}

func isNoise(line string) bool {
	lower := strings.ToLower(line)

	// Repeated PDF artifacts / legal boilerplate.
	noise := []string{
		"additional information on the next page",
		"please visit",
		"visit our website",
		"for more information",
		"terms and conditions",
		"billing rights",
		"what to do if",
		"your rights if",
	}

	for _, phrase := range noise {
		if strings.Contains(lower, phrase) {
			return true
		}
	}

	return false
}

func appendUnique(lines []string, line string) []string {
	if len(lines) == 0 || lines[len(lines)-1] != line {
		return append(lines, line)
	}

	return lines
}
