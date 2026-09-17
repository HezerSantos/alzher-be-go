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

	amountRe = regexp.MustCompile(
		`(?i)(?:` +
			`(?:minus|plus)\s*\$?\d[\d,]*\.\d{2}` +
			`|` +
			`[+-]?\s*\$?\d[\d,]*\.\d{2}` +
			`|` +
			`\(\s*\$?\d[\d,]*\.\d{2}\s*\)` +
			`|` +
			`(?:minus|plus)\s*\$?\d[\d,]*(?:\.\d{2})?` +
			`|` +
			`[+-]?\s*\$\d[\d,]*(?:\.\d{2})?` +
			`)`,
	)

	whitespaceRe = regexp.MustCompile(`\s+`)

	transactionRowRe = regexp.MustCompile(
		`(?i)^\s*` +
			`\d{1,2}[/-]\d{1,2}` +
			`\s+.+?\s+` +
			`(?:` +
			`(?:minus|plus)\s*\$?[\d,]+\.\d{2}` +
			`|` +
			`[-+]?\$?\(?[\d,]+\.\d{2}\)?` +
			`)` +
			`\s*$`,
	)

	transactionRowWithTwoDatesRe = regexp.MustCompile(
		`(?i)^\s*` +
			`(?:` +
			`\d{1,2}[/-]\d{1,2}` +
			`|` +
			`(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\s+\d{1,2}` +
			`)` +
			`\s+` +
			`(?:` +
			`\d{1,2}[/-]\d{1,2}` +
			`|` +
			`(?:jan|feb|mar|apr|may|jun|jul|aug|sep|sept|oct|nov|dec)[a-z]*\s+\d{1,2}` +
			`)` +
			`\s+.+?\s+` +
			`(?:` +
			`(?:minus|plus)\s*\$?[\d,]+\.\d{2}` +
			`|` +
			`[-+]?\$?\(?[\d,]+\.\d{2}\)?` +
			`)` +
			`\s*$`,
	)
)

func NormalizeStatementText(input string) string {
	lines := cleanLines(input)

	regions := findTransactionRegions(lines)

	var result []string

	// Add normal transaction regions first.
	for _, region := range regions {
		for _, line := range region {
			result = appendUnique(result, line)
		}
	}

	// Statement totals are independent of transaction regions.
	// This ensures Groq always has totals available for verification.
	for _, line := range lines {
		if isStatementTotal(line) {
			result = appendUnique(result, line)
		}
	}

	// Some PDFs collapse multiple transaction rows into one
	// giant line. Only preserve lines that contain multiple
	// actual date -> amount transaction patterns.
	for _, line := range lines {
		if isCollapsedTransactionLine(line) {
			result = appendUnique(result, line)
		}
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
	return isTransactionLine(line)
}

func isTransactionLine(line string) bool {
	line = strings.TrimSpace(line)

	if line == "" {
		return false
	}

	if transactionRowRe.MatchString(line) {
		return true
	}

	if transactionRowWithTwoDatesRe.MatchString(line) {
		return true
	}

	return false
}

func isCollapsedTransactionLine(line string) bool {
	line = strings.TrimSpace(line)

	if line == "" {
		return false
	}

	// Find all dates in the line.
	dates := dateRe.FindAllStringIndex(line, -1)

	// A collapsed transaction row needs at least
	// two transaction-looking dates.
	if len(dates) < 2 {
		return false
	}

	transactionCount := 0

	for i, date := range dates {
		end := len(line)

		// Only inspect the text between this date and
		// the next date. This prevents unrelated dates
		// later in a paragraph from being paired together.
		if i+1 < len(dates) {
			end = dates[i+1][0]
		}

		segment := line[date[1]:end]

		if amountRe.MatchString(segment) {
			transactionCount++
		}
	}

	// Require at least two actual date -> amount pairs.
	return transactionCount >= 2
}

func extractRegion(lines []string, start, end int) []string {
	end = min(end, len(lines))

	var result []string

	for i := start; i < end; i++ {
		line := lines[i]

		if isNoise(line) {
			continue
		}

		if isTransactionLine(line) {
			result = appendUnique(result, line)
			continue
		}

		if isStatementTotal(line) {
			result = appendUnique(result, line)
			continue
		}

		if isLikelyDescription(line) {
			result = appendUnique(result, line)
		}
	}

	return result
}

func isStatementTotal(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	if !amountRe.MatchString(line) {
		return false
	}

	if (strings.Contains(lower, "transactions") ||
		strings.Contains(lower, "purchases")) &&
		strings.Contains(lower, "+") {
		return true
	}

	if strings.HasPrefix(lower, "total transactions") ||
		strings.HasPrefix(lower, "total purchases") {
		return true
	}

	if strings.Contains(lower, "total fees") &&
		strings.Contains(lower, "total interest") {
		return true
	}

	return false
}

func isStatementMetadata(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	metadata := []string{
		"previous balance",
		"payments",
		"payments and credits",
		"other credits",
		"transactions +",
		"cash advances",
		"fees charged",
		"interest charged",
		"new balance",
		"credit limit",
		"available credit",
		"credit line",
		"minimum payment",
		"payment due date",
		"total fees",
		"total interest",
		"total cashback",
		"cashback bonus",
		"total points",
		"points redeemed",
		"earned this period",
		"redeemed this period",
		"new charges",
		"past due amount",
		"balance over the credit limit",
	}

	for _, phrase := range metadata {
		if strings.HasPrefix(lower, phrase) {
			return true
		}
	}

	return false
}

func isLikelyDescription(line string) bool {
	line = strings.TrimSpace(line)

	if line == "" {
		return false
	}

	if len(line) > 160 {
		return false
	}

	lower := strings.ToLower(line)

	headerNoise := []string{
		"account summary",
		"payment information",
		"payment and credits",
		"payments and credits",
		"payments, credits and adjustments",
		"transactions",
		"transactions continued",
		"standard purchases",
		"purchases",
		"fees",
		"fees charged",
		"interest",
		"interest charge",
		"account notifications",
		"rewards summary",
		"rewards",
		"information for you",
		"information for you continued",
		"account messages",
		"how you earn",
		"how do we calculate",
		"payment history",
		"purchase",
		"advances",
		"cash advances",
		"balance transfers",
		"statement date",
		"billing period",
		"billing cycle",
		"open to close date",
		"trans date",
		"post date",
		"sale date",
		"date of transaction",
		"merchant name or transaction description",
		"merchant category",
		"amount",
		"cardholder summary",
		"your account messages",
	}

	for _, phrase := range headerNoise {
		if lower == phrase || strings.HasPrefix(lower, phrase+" ") {
			return false
		}
	}

	metadataNoise := []string{
		"previous balance",
		"payments",
		"payments and credits",
		"payment due date",
		"minimum payment",
		"minimum payment due",
		"new balance",
		"credit limit",
		"credit line",
		"credit line available",
		"available credit",
		"available credit limit",
		"cash advance credit limit",
		"cash advance credit line",
		"available for cash",
		"available for cash advances",
		"cash advances",
		"cash advance",
		"balance transfers",
		"fees charged",
		"interest charged",
		"interest charge",
		"interest charges",
		"total fees",
		"total interest",
		"earned this period",
		"redeemed this period",
		"cashback bonus balance",
		"cashback bonus",
		"thankyou points",
		"thankyou points earned",
		"total points",
		"points redeemed",
		"previous points balance",
		"points available for redemption",
		"new charges",
		"past due amount",
		"balance over the credit limit",
	}

	for _, phrase := range metadataNoise {
		if strings.HasPrefix(lower, phrase) {
			return false
		}
	}

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
		"late payment warning",
		"for online and phone payments",
		"upcoming statement closing date",
		"if you make no additional charges",
		"you will pay off",
		"estimated total",
		"credit counseling",
		"call 888-",
		"call 1-",
		"cardmember since",
		"member since",
		"account ending in",
		"your fico",
		"score range",
		"score ingredients",
		"see your cardmember agreement",
		"cardmember agreement",
		"to make changes",
		"report immediately",
		"sending cash is not allowed",
		"processing of your allowable",
		"payments received",
		"paymentsreceived",
		"paymentto",
		"payment to",
		"discover may monitor",
		"the discover card is issued",
		"issued by discover bank",
		"for undeliverable mail only",
		"po box",
		"copyright",
		"how do i",
		"how can i",
		"your rights",
		"how is the interest charge",
		"how do we calculate",
		"do you assess a minimum interest",
		"we use a method called",
		"average daily balance",
		"prime rate",
		"libor",
		"when your apr",
		"apr will change",
		"activate your",
		"learn more",
		"visit ",
		"download our app",
		"mobile app",
		"online payments",
		"pay your bill",
		"see key factors",
		"see your score",
	}

	for _, phrase := range noise {
		if strings.Contains(lower, phrase) {
			return false
		}
	}

	if looksLikeAddressOrAccountInfo(line) {
		return false
	}

	fields := strings.Fields(line)

	if len(fields) == 1 {
		if amountRe.MatchString(line) {
			return false
		}

		if isNumericLike(line) {
			return false
		}
	}

	if looksLikeRateLine(lower) {
		return false
	}

	return true
}

func looksLikeAddressOrAccountInfo(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))

	if strings.Contains(lower, "account") ||
		strings.Contains(lower, "hezer santos") ||
		strings.Contains(lower, "card ending") ||
		strings.Contains(lower, "member since") {
		return true
	}

	addressPrefixes := []string{
		"po box",
		"p.o. box",
	}

	for _, prefix := range addressPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

func isNumericLike(line string) bool {
	line = strings.TrimSpace(line)

	if line == "" {
		return false
	}

	for _, r := range line {
		if (r < '0' || r > '9') &&
			r != '.' &&
			r != ',' &&
			r != '/' &&
			r != '-' &&
			r != '+' &&
			r != '$' {
			return false
		}
	}

	return true
}

func looksLikeRateLine(lower string) bool {
	rateTerms := []string{
		"apr",
		"prime rate",
		"libor",
		"periodic rate",
		"introductory rate",
		"standard purch",
		"standard adv",
		"cash advances",
		"balance transfers",
	}

	for _, term := range rateTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}

	return false
}

func isNoise(line string) bool {
	lower := strings.ToLower(line)

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
