package pdf

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// IntelligentParser parses extracted text to identify structured data
type IntelligentParser struct {
	// Configurable patterns for different document types
	CustomerPatterns  []*regexp.Regexp
	DatePatterns      []*regexp.Regexp
	InvoiceNoPatterns []*regexp.Regexp
	TotalPatterns     []*regexp.Regexp
	DiscountPatterns  []*regexp.Regexp
	ItemPatterns      []*regexp.Regexp
}

// NewIntelligentParser creates a new parser with pre-configured patterns
func NewIntelligentParser() *IntelligentParser {
	return &IntelligentParser{
		CustomerPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:kunde|customer|rechnung\s+an|bill\s+to|empfänger|client|auftraggeber)[\s:]+(.+?)(?:\n|$)`),
			regexp.MustCompile(`(?i)(?:firma|company|unternehmen)[\s:]+(.+?)(?:\n|$)`),
		},
		DatePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:datum|date|vom|on|issued)[\s:]*(\d{1,2})[\./-](\d{1,2})[\./-](\d{2,4})`),
			regexp.MustCompile(`(\d{1,2})[\./-](\d{1,2})[\./-](\d{4})`),
		},
		InvoiceNoPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:rechnung|invoice|angebot|offer|offerte|quotation)[\s#:№-]+([A-Z0-9\-_/]+)`),
			regexp.MustCompile(`(?i)(?:nr|no|number|nummer)[\s#:\.]+([A-Z0-9\-_/]+)`),
		},
		TotalPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:gesamt|total|summe|sum|endsumme|gesamtbetrag)[\s:]*€?\s*([0-9.,]+)`),
			regexp.MustCompile(`(?i)(?:zu\s+zahlen|to\s+pay|amount\s+due)[\s:]*€?\s*([0-9.,]+)`),
		},
		DiscountPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(?:rabatt|discount|nachlass|skonto)[\s:]*€?\s*([0-9.,]+)`),
			regexp.MustCompile(`(?i)(?:abzug|reduction|ermäßigung)[\s:]*€?\s*([0-9.,]+)`),
		},
		ItemPatterns: []*regexp.Regexp{
			// Pattern 0: Pos | Qty | Description | Unit Price | Discount | Total
			regexp.MustCompile(`^(\d+)\s+(\d+)\s*x?\s+(.+?)\s+€?\s*([0-9.,]+)\s+([+-]?[0-9.,]+)\s*(%|prozent|percent)?\s+€?\s*([0-9.,]+)$`),
			// Pattern 1: Pos | Qty | Description | Unit Price | Total
			regexp.MustCompile(`(\d+)\s+(\d+)\s*x?\s+(.+?)\s+€?\s*([0-9.,]+)\s+€?\s*([0-9.,]+)`),
			// Pattern 2: Description | Qty | Price | Total
			regexp.MustCompile(`^(.{20,}?)\s+(\d+)\s+([0-9.,]+)\s+([0-9.,]+)$`),
			// Pattern 3: Simple format with position
			regexp.MustCompile(`^(\d+)[\.\)]\s+(.+?)\s+(\d+)\s*(?:x|stk|pcs|pieces)?\s+€?\s*([0-9.,]+)`),
		},
	}
}

// ParsedDocumentType represents the detected document type
type ParsedDocumentType string

const (
	DocTypeInvoice  ParsedDocumentType = "invoice"
	DocTypeOffer    ParsedDocumentType = "offer"
	DocTypeOrder    ParsedDocumentType = "order"
	DocTypeDelivery ParsedDocumentType = "delivery"
	DocTypeUnknown  ParsedDocumentType = "unknown"
)

// ParsedDocument represents fully parsed document data
type ParsedDocument struct {
	DocumentType    ParsedDocumentType
	CustomerName    string
	CustomerID      *int
	DocumentNumber  string
	DocumentDate    time.Time
	ParsedTotal     float64 // Subtotal before discount (Zwischensumme netto)
	DiscountAmount  float64 // Total discount amount
	DiscountPercent float64 // Discount percentage
	TotalAmount     float64 // Final amount after discount (Gesamtbetrag)
	Items           []ParsedItem
	RawSections     map[string]string // Store raw sections for reference
	ConfidenceScore float64
	Metadata        map[string]interface{}
}

// ParsedItem represents a parsed line item with type detection
type ParsedItem struct {
	LineNumber       int
	RawText          string
	DetectedType     ItemType
	ProductName      string
	Quantity         int
	UnitPrice        float64
	LineTotal        float64
	ConfidenceScore  float64
	AlternativeNames []string // Alternative product name interpretations
}

// ItemType represents the detected type of item
type ItemType string

const (
	ItemTypeProduct  ItemType = "product"
	ItemTypeCustomer ItemType = "customer"
	ItemTypeDiscount ItemType = "discount"
	ItemTypeHeader   ItemType = "header"
	ItemTypeOther    ItemType = "other"
)

// ParseDocument intelligently parses extracted text
func (p *IntelligentParser) ParseDocument(rawText string) (*ParsedDocument, error) {
	doc := &ParsedDocument{
		Items:       []ParsedItem{},
		RawSections: make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}

	lines := strings.Split(rawText, "\n")

	// 1. Detect document type
	doc.DocumentType = p.detectDocumentType(rawText)

	// 2. Extract customer information
	doc.CustomerName = p.extractCustomerName(lines)

	// 3. Extract document number
	doc.DocumentNumber = p.extractDocumentNumber(lines)

	// 4. Extract document date
	doc.DocumentDate = p.extractDocumentDate(lines)

	// 5. Extract total amount
	doc.TotalAmount = p.extractTotalAmount(lines)

	// 6. Extract discount
	doc.DiscountAmount = p.extractDiscount(lines, doc.TotalAmount)

	// 7. Parse line items
	items, err := p.parseLineItems(lines)
	if err == nil {
		doc.Items = items
	}

	// 8. Calculate confidence score
	doc.ConfidenceScore = p.calculateDocumentConfidence(doc)

	return doc, nil
}

// detectDocumentType detects the type of document from text
func (p *IntelligentParser) detectDocumentType(text string) ParsedDocumentType {
	textLower := strings.ToLower(text)

	invoiceKeywords := []string{"rechnung", "invoice", "faktura"}
	offerKeywords := []string{"angebot", "offer", "offerte", "quotation", "quote"}
	orderKeywords := []string{"bestellung", "order", "auftrag"}
	deliveryKeywords := []string{"lieferschein", "delivery", "versand"}

	for _, kw := range invoiceKeywords {
		if strings.Contains(textLower, kw) {
			return DocTypeInvoice
		}
	}
	for _, kw := range offerKeywords {
		if strings.Contains(textLower, kw) {
			return DocTypeOffer
		}
	}
	for _, kw := range orderKeywords {
		if strings.Contains(textLower, kw) {
			return DocTypeOrder
		}
	}
	for _, kw := range deliveryKeywords {
		if strings.Contains(textLower, kw) {
			return DocTypeDelivery
		}
	}

	return DocTypeUnknown
}

// extractCustomerName extracts customer name from lines
func (p *IntelligentParser) extractCustomerName(lines []string) string {
	for _, line := range lines {
		for _, pattern := range p.CustomerPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				name := strings.TrimSpace(matches[1])
				// Clean up the name
				name = p.cleanCustomerName(name)
				if len(name) > 3 {
					return name
				}
			}
		}
	}
	return ""
}

// cleanCustomerName cleans up extracted customer name
func (p *IntelligentParser) cleanCustomerName(name string) string {
	// Remove common suffixes
	name = strings.TrimSpace(name)
	// Remove trailing punctuation
	name = strings.TrimRight(name, ".,;:")
	// Remove extra whitespace
	name = regexp.MustCompile(`\s+`).ReplaceAllString(name, " ")
	return name
}

// extractDocumentNumber extracts document/invoice number
func (p *IntelligentParser) extractDocumentNumber(lines []string) string {
	for _, line := range lines {
		for _, pattern := range p.InvoiceNoPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				number := strings.TrimSpace(matches[1])
				if len(number) >= 3 && len(number) <= 50 {
					return number
				}
			}
		}
	}
	return ""
}

// extractDocumentDate extracts document date
func (p *IntelligentParser) extractDocumentDate(lines []string) time.Time {
	for _, line := range lines {
		for _, pattern := range p.DatePatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) >= 4 {
				day, _ := strconv.Atoi(matches[1])
				month, _ := strconv.Atoi(matches[2])
				year, _ := strconv.Atoi(matches[3])

				// Handle 2-digit years
				if year < 100 {
					if year > 50 {
						year += 1900
					} else {
						year += 2000
					}
				}

				// Validate date
				if day >= 1 && day <= 31 && month >= 1 && month <= 12 && year >= 2000 && year <= 2100 {
					return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
				}
			}
		}
	}
	return time.Time{}
}

// extractTotalAmount extracts total amount
func (p *IntelligentParser) extractTotalAmount(lines []string) float64 {
	var maxAmount float64 = 0

	for _, line := range lines {
		for _, pattern := range p.TotalPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				amount := p.parseAmount(matches[1])
				if amount > maxAmount {
					maxAmount = amount
				}
			}
		}
	}

	return maxAmount
}

// extractDiscount extracts discount amount
func (p *IntelligentParser) extractDiscount(lines []string, totalAmount float64) float64 {
	for idx, line := range lines {
		lineLower := strings.ToLower(line)
		if !containsKeyword(lineLower, discountKeywords) {
			continue
		}
		if token, ok := findDecimalAmountToken(line); ok {
			amount := p.parseAmount(token)
			if amount < 0 {
				amount = -amount
			}
			if amount > 0 {
				return amount
			}
		}
		if pct, ok := findPercentage(line); ok && totalAmount > 0 {
			amount := totalAmount * pct / 100
			if amount > 0 {
				return amount
			}
		}
		if token, ok := findAmountInSubsequentLines(lines, idx, 3); ok {
			amount := p.parseAmount(token)
			if amount < 0 {
				amount = -amount
			}
			if amount > 0 {
				return amount
			}
		}
	}
	return 0
}

func findAmountInSubsequentLines(lines []string, startIdx, lookahead int) (string, bool) {
	for j := 1; j <= lookahead && startIdx+j < len(lines); j++ {
		candidate := strings.TrimSpace(lines[startIdx+j])
		if candidate == "" {
			continue
		}
		if token, ok := findDecimalAmountToken(candidate); ok {
			return token, true
		}
	}
	return "", false
}

// parseLineItems parses line items from document
func (p *IntelligentParser) parseLineItems(lines []string) ([]ParsedItem, error) {
	var items []ParsedItem
	sequenceNumber := 0

	tableStart, tableEnd := p.detectTableBoundaries(lines)
	discountColumnActive := false
	var pendingDescriptor *lineDescriptor

	for i := tableStart; i <= tableEnd && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			if discountColumnActive {
				discountColumnActive = false
			}
			continue
		}

		lineLower := strings.ToLower(line)
		if p.isHeaderLine(line) {
			if strings.Contains(lineLower, "rabatt") {
				discountColumnActive = true
			}
			continue
		}

		if discountColumnActive {
			if descriptor := p.extractLineDescriptor(line); descriptor != nil {
				pendingDescriptor = descriptor
				continue
			}
			if strings.HasPrefix(lineLower, "zwischensumme") || strings.HasPrefix(lineLower, "gesamtbetrag") || strings.HasPrefix(lineLower, "vielen dank") || strings.HasPrefix(lineLower, "wir ") {
				discountColumnActive = false
				pendingDescriptor = nil
			} else {
				if item := p.parseDiscountColumnLine(line, i, pendingDescriptor); item != nil {
					pendingDescriptor = nil
					if item.LineNumber == 0 {
						item.LineNumber = sequenceNumber + 1
					}
					items = append(items, *item)
					sequenceNumber++
					continue
				}
			}
		}

		if strings.HasPrefix(lineLower, "übertrag") {
			continue
		}

		if len(line) < 5 {
			continue
		}

		matched := false
		for _, pattern := range p.ItemPatterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 0 {
				item := p.parseItemFromMatches(matches, sequenceNumber, line)
				if item != nil {
					pendingDescriptor = nil
					items = append(items, *item)
					sequenceNumber++
					matched = true
					break
				}
			}
		}
		if matched {
			continue
		}

		if item := p.parseItemIntelligently(line, sequenceNumber); item != nil {
			pendingDescriptor = nil
			items = append(items, *item)
			sequenceNumber++
		}
	}

	return items, nil
}

// detectTableBoundaries detects where the item table starts and ends
func (p *IntelligentParser) detectTableBoundaries(lines []string) (int, int) {
	start := -1
	end := len(lines) - 1

	// Look for table headers
	headerKeywords := []string{"pos", "menge", "beschreibung", "preis", "qty", "description", "price", "artikel"}

	for i, line := range lines {
		lineLower := strings.ToLower(line)
		matchCount := 0
		for _, kw := range headerKeywords {
			if strings.Contains(lineLower, kw) {
				matchCount++
			}
		}
		if matchCount >= 2 {
			start = i + 1 // Start after header
			break
		}
	}

	if start == -1 {
		start = 0
	}

	// Look for table end (usually marked by "Summe", "Total", etc.)
	endKeywords := []string{"summe", "total", "gesamt", "subtotal", "netto", "brutto"}
	for i := start; i < len(lines); i++ {
		lineLower := strings.ToLower(lines[i])
		for _, kw := range endKeywords {
			if strings.Contains(lineLower, kw) {
				end = i - 1
				return start, end
			}
		}
	}

	return start, end
}

func insertAlphaNumericSpacing(line string) string {
	var builder strings.Builder
	prevType := 0
	for _, r := range line {
		currType := 0
		switch {
		case unicode.IsLetter(r):
			currType = 1
		case unicode.IsDigit(r) || r == ',' || r == '.':
			currType = 2
		}
		if builder.Len() > 0 && currType != 0 && prevType != 0 && currType != prevType {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
		if currType == 0 {
			prevType = 0
		} else {
			prevType = currType
		}
	}
	return builder.String()
}

var defaultUnitTokens = []string{
	"stück", "stk", "st", "set", "sets", "pcs", "person", "personen",
	"tag", "tage", "std", "stunden", "hour", "hours", "case", "crew",
}

func isUnitToken(token string) bool {
	token = strings.TrimSpace(strings.ToLower(token))
	if token == "" {
		return false
	}
	for _, kw := range defaultUnitTokens {
		if token == kw {
			return true
		}
	}
	return false
}

type lineDescriptor struct {
	lineNumber  int
	description string
}

var descriptorLineRegex = regexp.MustCompile(`^(\d+)\s+(.+)$`)

func (p *IntelligentParser) extractLineDescriptor(line string) *lineDescriptor {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}
	if amountTokenRegex.FindStringIndex(trimmed) != nil {
		return nil
	}
	matches := descriptorLineRegex.FindStringSubmatch(trimmed)
	if len(matches) != 3 {
		return nil
	}
	lineNumber, err := strconv.Atoi(strings.TrimSuffix(matches[1], "."))
	if err != nil || lineNumber <= 0 {
		return nil
	}
	description := strings.TrimSpace(matches[2])
	if description == "" {
		return nil
	}
	return &lineDescriptor{
		lineNumber:  lineNumber,
		description: description,
	}
}

func (p *IntelligentParser) parseDiscountColumnLine(rawLine string, lineNum int, descriptor *lineDescriptor) *ParsedItem {
	normalized := insertAlphaNumericSpacing(rawLine)
	amountMatches := amountTokenRegex.FindAllStringIndex(normalized, -1)
	if len(amountMatches) < 3 {
		return nil
	}

	totalIdx := amountMatches[len(amountMatches)-1]
	discountIdx := amountMatches[len(amountMatches)-2]
	unitIdx := amountMatches[len(amountMatches)-3]

	totalStr := normalized[totalIdx[0]:totalIdx[1]]
	discountStr := normalized[discountIdx[0]:discountIdx[1]]
	unitStr := normalized[unitIdx[0]:unitIdx[1]]

	prefix := strings.TrimSpace(normalized[:unitIdx[0]])
	if prefix == "" && descriptor == nil {
		return nil
	}

	tokens := strings.Fields(prefix)

	lineNumber := 0
	tokenStart := 0
	if descriptor == nil && len(tokens) > 1 {
		if value, err := strconv.Atoi(strings.TrimSuffix(tokens[0], ".")); err == nil && value > 0 {
			lineNumber = value
			tokenStart = 1
		}
	}
	if lineNumber == 0 && descriptor != nil {
		lineNumber = descriptor.lineNumber
	}
	if lineNumber == 0 {
		lineNumber = lineNum + 1
	}

	quantity := 0
	qtyIndex := -1
	for idx := len(tokens) - 1; idx >= tokenStart; idx-- {
		token := strings.TrimSpace(strings.TrimSuffix(tokens[idx], "."))
		if token == "" || isUnitToken(token) {
			continue
		}
		if val, err := strconv.Atoi(token); err == nil && val > 0 {
			quantity = val
			qtyIndex = idx
			break
		}
	}
	if quantity == 0 {
		if descriptor == nil {
			return nil
		}
		quantity = 1
		qtyIndex = len(tokens)
	}

	descriptionTokens := []string{}
	if qtyIndex > tokenStart {
		descriptionTokens = tokens[tokenStart:qtyIndex]
	}
	description := strings.TrimSpace(strings.Join(descriptionTokens, " "))
	if description == "" && descriptor != nil {
		description = descriptor.description
	}
	if description == "" {
		return nil
	}

	unitPrice := p.parseAmount(unitStr)
	discountPercent := p.parseAmount(discountStr)
	total := p.parseAmount(totalStr)
	if unitPrice <= 0 {
		return nil
	}

	linePrice := unitPrice * float64(quantity)
	if discountPercent > 0 {
		discountAmount := p.calculateLineDiscount(linePrice, discountPercent, "", true)
		lineTotal := linePrice - discountAmount
		if lineTotal < 0 {
			lineTotal = 0
		}
		total = lineTotal
	}

	if total == 0 && linePrice > 0 && discountPercent == 0 {
		total = linePrice
	}

	item := &ParsedItem{
		LineNumber:      lineNumber,
		RawText:         rawLine,
		DetectedType:    ItemTypeProduct,
		ProductName:     description,
		Quantity:        quantity,
		UnitPrice:       unitPrice,
		LineTotal:       total,
		ConfidenceScore: 93.0,
	}

	return item
}

// isHeaderLine checks if a line is a table header
func (p *IntelligentParser) isHeaderLine(line string) bool {
	lineLower := strings.ToLower(line)
	headerKeywords := []string{"pos", "menge", "beschreibung", "preis", "qty", "description", "price", "total", "gesamt"}

	matchCount := 0
	for _, kw := range headerKeywords {
		if strings.Contains(lineLower, kw) {
			matchCount++
		}
	}

	return matchCount >= 2
}

// parseItemFromMatches parses an item from regex matches
func (p *IntelligentParser) parseItemFromMatches(matches []string, lineNum int, rawLine string) *ParsedItem {
	if len(matches) < 4 {
		return nil
	}

	item := &ParsedItem{
		LineNumber:      lineNum + 1,
		RawText:         rawLine,
		DetectedType:    ItemTypeProduct,
		ConfidenceScore: 70.0,
	}

	// Pattern-specific parsing
	switch len(matches) {
	case 8: // Pos, qty, desc, unit, discount, indicator, total
		item.Quantity, _ = strconv.Atoi(matches[2])
		if item.Quantity <= 0 {
			item.Quantity = 1
		}
		item.ProductName = strings.TrimSpace(matches[3])
		item.UnitPrice = p.parseAmount(matches[4])
		discountValue := math.Abs(p.parseAmount(matches[5]))
		discountIndicator := strings.TrimSpace(strings.ToLower(matches[6]))
		rawTotal := p.parseAmount(matches[7])
		item.LineTotal = rawTotal

		linePrice := item.UnitPrice * float64(item.Quantity)
		if discountValue > 0 && linePrice > 0 {
			discountAmount := p.calculateLineDiscount(linePrice, discountValue, discountIndicator, true)
			if discountAmount > 0 {
				discountedTotal := linePrice - discountAmount
				if discountedTotal < 0 {
					discountedTotal = 0
				}
				if rawTotal == 0 || math.Abs(rawTotal-linePrice) < 0.01 || discountedTotal < rawTotal-0.01 {
					item.LineTotal = discountedTotal
				}
			}
		}

		if item.LineTotal <= 0 && linePrice > 0 {
			item.LineTotal = linePrice
		}

		item.ConfidenceScore = 92.0
	case 6: // Full pattern: pos, qty, desc, unit price, total
		item.Quantity, _ = strconv.Atoi(matches[2])
		item.ProductName = strings.TrimSpace(matches[3])
		item.UnitPrice = p.parseAmount(matches[4])
		item.LineTotal = p.parseAmount(matches[5])
		item.ConfidenceScore = 90.0
	case 5: // desc, qty, unit price, total
		item.ProductName = strings.TrimSpace(matches[1])
		item.Quantity, _ = strconv.Atoi(matches[2])
		item.UnitPrice = p.parseAmount(matches[3])
		item.LineTotal = p.parseAmount(matches[4])
		item.ConfidenceScore = 85.0
	case 4: // pos, desc, qty, price
		item.ProductName = strings.TrimSpace(matches[2])
		item.Quantity, _ = strconv.Atoi(matches[3])
		item.UnitPrice = p.parseAmount(matches[4])
		item.LineTotal = item.UnitPrice * float64(item.Quantity)
		item.ConfidenceScore = 80.0
	}

	// Validate item
	if item.ProductName == "" || item.Quantity == 0 {
		return nil
	}

	return item
}

// parseItemIntelligently attempts to parse a line intelligently
func (p *IntelligentParser) parseItemIntelligently(line string, lineNum int) *ParsedItem {
	// Look for patterns: numbers, text, prices
	words := strings.Fields(line)
	if len(words) < 3 {
		return nil
	}

	item := &ParsedItem{
		LineNumber:      lineNum + 1,
		RawText:         line,
		DetectedType:    ItemTypeProduct,
		ConfidenceScore: 60.0,
		Quantity:        1, // Default
	}

	// Extract numbers (potential quantities and prices)
	var numbers []float64
	var textParts []string

	for _, word := range words {
		// Check if it's a number
		if num := p.parseAmount(word); num > 0 {
			numbers = append(numbers, num)
		} else if !p.isNumeric(word) {
			textParts = append(textParts, word)
		}
	}

	// Product name is the text part
	if len(textParts) > 0 {
		item.ProductName = strings.Join(textParts, " ")
	}

	// Assign numbers to quantity/price/total
	switch len(numbers) {
	case 2:
		// Only qty + total present (no separate unit price column).
		// Bug guard: numbers[-2] == numbers[0] == qty → would put qty into UnitPrice.
		// Instead derive unit price from total / qty.
		if numbers[0] > 0 && numbers[0] < 100 && numbers[0] == float64(int(numbers[0])) {
			item.Quantity = int(numbers[0])
			item.LineTotal = numbers[1]
			item.UnitPrice = numbers[1] / numbers[0]
		} else {
			item.UnitPrice = numbers[0]
			item.LineTotal = numbers[1]
		}
	case 3:
		// qty + price + total (standard)
		if numbers[0] < 100 && numbers[0] == float64(int(numbers[0])) {
			item.Quantity = int(numbers[0])
		}
		item.UnitPrice = numbers[1]
		item.LineTotal = numbers[2]
	default:
		// 4+ numbers: could be [pos, qty, price, total] or similar.
		// Use qty × price ≈ total to identify actual quantity field.
		price := numbers[len(numbers)-2]
		total := numbers[len(numbers)-1]
		item.UnitPrice = price
		item.LineTotal = total
		if total > 0 && price > 0 {
			// Check second-from-price number first (position of potential qty)
			candidateIdx := len(numbers) - 3
			if candidateIdx >= 0 {
				qtyCandidate := int(numbers[candidateIdx])
				if math.Abs(float64(qtyCandidate)*price-total)/total < 0.02 {
					item.Quantity = qtyCandidate
				} else if numbers[0] < 100 && numbers[0] == float64(int(numbers[0])) {
					item.Quantity = int(numbers[0])
				}
			}
		} else if numbers[0] < 100 && numbers[0] == float64(int(numbers[0])) {
			item.Quantity = int(numbers[0])
		}
	}

	if item.ProductName == "" {
		return nil
	}

	return item
}

func (p *IntelligentParser) calculateLineDiscount(linePrice, discountValue float64, indicator string, forcePercent bool) float64 {
	if linePrice <= 0 || discountValue <= 0 {
		return 0
	}

	indicator = strings.TrimSpace(strings.ToLower(indicator))
	if forcePercent || indicator != "" {
		if forcePercent || strings.Contains(indicator, "%") || strings.Contains(indicator, "prozent") || strings.Contains(indicator, "percent") {
			return linePrice * (discountValue / 100)
		}
	}

	if forcePercent {
		return linePrice * (discountValue / 100)
	}

	if discountValue > 0 && discountValue < 1 {
		return linePrice * discountValue
	}

	if discountValue <= 100 {
		return linePrice * (discountValue / 100)
	}

	if discountValue > linePrice {
		return linePrice
	}

	return discountValue
}

// parseAmount parses a monetary amount from string
func (p *IntelligentParser) parseAmount(s string) float64 {
	// Remove currency symbols and whitespace
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.ReplaceAll(s, "\u00a0", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)

	// Handle European format (1.234,56)
	if strings.Count(s, ".") > 0 && strings.Count(s, ",") > 0 {
		// Mixed format - remove thousand separators
		if strings.LastIndex(s, ",") > strings.LastIndex(s, ".") {
			// European: 1.234,56
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// American: 1,234.56
			s = strings.ReplaceAll(s, ",", "")
		}
	} else if strings.Count(s, ",") > 0 {
		// Only commas - could be thousand sep or decimal
		if strings.LastIndex(s, ",") == len(s)-3 {
			// Likely decimal: 123,45
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// Likely thousand separator: 1,234
			s = strings.ReplaceAll(s, ",", "")
		}
	}

	amount, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return amount
}

// isNumeric checks if a string is numeric
func (p *IntelligentParser) isNumeric(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != ',' {
			return false
		}
	}
	return true
}

// calculateDocumentConfidence calculates overall document parsing confidence
func (p *IntelligentParser) calculateDocumentConfidence(doc *ParsedDocument) float64 {
	score := 0.0
	maxScore := 7.0

	if doc.DocumentType != DocTypeUnknown {
		score += 1.0
	}
	if doc.CustomerName != "" {
		score += 1.0
	}
	if doc.DocumentNumber != "" {
		score += 1.5
	}
	if !doc.DocumentDate.IsZero() {
		score += 1.0
	}
	if doc.TotalAmount > 0 {
		score += 1.0
	}
	if len(doc.Items) > 0 {
		score += 1.5
	}

	return (score / maxScore) * 100.0
}

// DetectItemType detects the type of item from its content
func (p *IntelligentParser) DetectItemType(itemText string) ItemType {
	textLower := strings.ToLower(itemText)

	// Check for customer indicators
	customerKeywords := []string{"kunde", "customer", "firma", "company"}
	for _, kw := range customerKeywords {
		if strings.Contains(textLower, kw) {
			return ItemTypeCustomer
		}
	}

	// Check for discount indicators
	discountKeywords := []string{"rabatt", "discount", "nachlass", "skonto"}
	for _, kw := range discountKeywords {
		if strings.Contains(textLower, kw) {
			return ItemTypeDiscount
		}
	}

	// Check for header indicators
	if p.isHeaderLine(itemText) {
		return ItemTypeHeader
	}

	// Default to product
	if len(itemText) > 5 && (strings.Contains(textLower, "led") ||
		strings.Contains(textLower, "par") ||
		strings.Contains(textLower, "light") ||
		strings.Contains(textLower, "mikro") ||
		strings.Contains(textLower, "cable")) {
		return ItemTypeProduct
	}

	return ItemTypeOther
}
