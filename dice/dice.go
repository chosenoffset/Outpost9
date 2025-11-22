// Package dice provides a flexible dice rolling and expression evaluation system
// for tabletop-style RPG mechanics. It supports standard dice notation (3d6),
// arithmetic operations, keep highest/lowest, and comparison functions.
package dice

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// RollResult contains the result of a dice roll or expression evaluation
type RollResult struct {
	Total      int    // Final computed value
	Rolls      []int  // Individual die rolls (if applicable)
	Expression string // Original expression
	Breakdown  string // Human-readable breakdown of the roll
}

// Roller handles dice rolling with a configurable random source
type Roller struct {
	rng *rand.Rand
}

// NewRoller creates a new Roller with the given random source
func NewRoller(rng *rand.Rand) *Roller {
	return &Roller{rng: rng}
}

// Roll evaluates a dice expression and returns the result
// Supported syntax:
//   - Basic dice: "3d6" (roll 3 six-sided dice)
//   - Arithmetic: "3d6+5", "2d8-2", "3d6*2", "2d10/2"
//   - Keep highest: "4d6kh3" or "4d6k3" (roll 4d6, keep highest 3)
//   - Keep lowest: "4d6kl3" (roll 4d6, keep lowest 3)
//   - Drop highest: "4d6dh1" (roll 4d6, drop highest 1)
//   - Drop lowest: "4d6dl1" (roll 4d6, drop lowest 1)
//   - Constants: "5", "10"
//   - Parentheses: "(2d6+3)*2"
func (r *Roller) Roll(expression string) (*RollResult, error) {
	expr := strings.TrimSpace(strings.ToLower(expression))
	if expr == "" {
		return nil, fmt.Errorf("empty expression")
	}

	result := &RollResult{
		Expression: expression,
	}

	total, breakdown, rolls, err := r.evaluate(expr)
	if err != nil {
		return nil, err
	}

	result.Total = total
	result.Breakdown = breakdown
	result.Rolls = rolls

	return result, nil
}

// RollMultiple rolls the same expression multiple times and returns all results
func (r *Roller) RollMultiple(expression string, count int) ([]*RollResult, error) {
	results := make([]*RollResult, count)
	for i := 0; i < count; i++ {
		result, err := r.Roll(expression)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

// Best rolls the expression multiple times and returns the highest result
func (r *Roller) Best(expression string, count int) (*RollResult, error) {
	results, err := r.RollMultiple(expression, count)
	if err != nil {
		return nil, err
	}

	best := results[0]
	for _, result := range results[1:] {
		if result.Total > best.Total {
			best = result
		}
	}
	return best, nil
}

// Worst rolls the expression multiple times and returns the lowest result
func (r *Roller) Worst(expression string, count int) (*RollResult, error) {
	results, err := r.RollMultiple(expression, count)
	if err != nil {
		return nil, err
	}

	worst := results[0]
	for _, result := range results[1:] {
		if result.Total < worst.Total {
			worst = result
		}
	}
	return worst, nil
}

// evaluate parses and evaluates an expression, returning total, breakdown, and rolls
func (r *Roller) evaluate(expr string) (int, string, []int, error) {
	// Remove all whitespace
	expr = strings.ReplaceAll(expr, " ", "")

	// Handle parentheses first
	for strings.Contains(expr, "(") {
		start := strings.LastIndex(expr, "(")
		end := strings.Index(expr[start:], ")") + start
		if end <= start {
			return 0, "", nil, fmt.Errorf("mismatched parentheses in expression: %s", expr)
		}

		inner := expr[start+1 : end]
		innerTotal, innerBreakdown, _, err := r.evaluate(inner)
		if err != nil {
			return 0, "", nil, err
		}

		// Replace the parenthesized expression with its result
		expr = expr[:start] + strconv.Itoa(innerTotal) + expr[end+1:]
		_ = innerBreakdown // We lose some breakdown detail with parens
	}

	// Handle addition and subtraction (lowest precedence)
	// Find the rightmost + or - that's not part of a dice expression
	for i := len(expr) - 1; i >= 0; i-- {
		if (expr[i] == '+' || expr[i] == '-') && i > 0 {
			// Make sure this isn't the start of the expression
			left := expr[:i]
			right := expr[i+1:]
			op := string(expr[i])

			leftTotal, leftBreakdown, leftRolls, err := r.evaluate(left)
			if err != nil {
				return 0, "", nil, err
			}

			rightTotal, rightBreakdown, rightRolls, err := r.evaluate(right)
			if err != nil {
				return 0, "", nil, err
			}

			var total int
			if op == "+" {
				total = leftTotal + rightTotal
			} else {
				total = leftTotal - rightTotal
			}

			breakdown := fmt.Sprintf("%s %s %s", leftBreakdown, op, rightBreakdown)
			rolls := append(leftRolls, rightRolls...)
			return total, breakdown, rolls, nil
		}
	}

	// Handle multiplication and division
	for i := len(expr) - 1; i >= 0; i-- {
		if expr[i] == '*' || expr[i] == '/' {
			left := expr[:i]
			right := expr[i+1:]
			op := string(expr[i])

			leftTotal, leftBreakdown, leftRolls, err := r.evaluate(left)
			if err != nil {
				return 0, "", nil, err
			}

			rightTotal, rightBreakdown, rightRolls, err := r.evaluate(right)
			if err != nil {
				return 0, "", nil, err
			}

			var total int
			if op == "*" {
				total = leftTotal * rightTotal
			} else {
				if rightTotal == 0 {
					return 0, "", nil, fmt.Errorf("division by zero")
				}
				total = leftTotal / rightTotal
			}

			breakdown := fmt.Sprintf("%s %s %s", leftBreakdown, op, rightBreakdown)
			rolls := append(leftRolls, rightRolls...)
			return total, breakdown, rolls, nil
		}
	}

	// At this point, expr should be either a dice notation or a constant
	return r.evaluateTerm(expr)
}

// diceRegex matches dice notation like "3d6", "4d6kh3", "2d8dl1"
var diceRegex = regexp.MustCompile(`^(\d+)d(\d+)(?:(kh?|kl|dh|dl)(\d+))?$`)

// evaluateTerm evaluates a single term (dice or constant)
func (r *Roller) evaluateTerm(term string) (int, string, []int, error) {
	term = strings.TrimSpace(term)

	// Check if it's a constant
	if num, err := strconv.Atoi(term); err == nil {
		return num, strconv.Itoa(num), nil, nil
	}

	// Try to parse as dice notation
	matches := diceRegex.FindStringSubmatch(term)
	if matches == nil {
		return 0, "", nil, fmt.Errorf("invalid term: %s", term)
	}

	numDice, _ := strconv.Atoi(matches[1])
	sides, _ := strconv.Atoi(matches[2])
	modifier := matches[3]
	modValue := 0
	if matches[4] != "" {
		modValue, _ = strconv.Atoi(matches[4])
	}

	if numDice <= 0 || sides <= 0 {
		return 0, "", nil, fmt.Errorf("invalid dice specification: %s", term)
	}

	// Roll all the dice
	rolls := make([]int, numDice)
	for i := 0; i < numDice; i++ {
		rolls[i] = r.rng.Intn(sides) + 1
	}

	// Apply keep/drop modifiers
	keptRolls := rolls
	breakdown := fmt.Sprintf("[%s]", joinInts(rolls, ", "))

	if modifier != "" && modValue > 0 {
		sorted := make([]int, len(rolls))
		copy(sorted, rolls)
		sort.Ints(sorted)

		switch modifier {
		case "kh", "k": // Keep highest
			if modValue < len(sorted) {
				keptRolls = sorted[len(sorted)-modValue:]
				breakdown = fmt.Sprintf("[%s] kh%d → [%s]", joinInts(rolls, ", "), modValue, joinInts(keptRolls, ", "))
			}
		case "kl": // Keep lowest
			if modValue < len(sorted) {
				keptRolls = sorted[:modValue]
				breakdown = fmt.Sprintf("[%s] kl%d → [%s]", joinInts(rolls, ", "), modValue, joinInts(keptRolls, ", "))
			}
		case "dh": // Drop highest
			if modValue < len(sorted) {
				keptRolls = sorted[:len(sorted)-modValue]
				breakdown = fmt.Sprintf("[%s] dh%d → [%s]", joinInts(rolls, ", "), modValue, joinInts(keptRolls, ", "))
			}
		case "dl": // Drop lowest
			if modValue < len(sorted) {
				keptRolls = sorted[modValue:]
				breakdown = fmt.Sprintf("[%s] dl%d → [%s]", joinInts(rolls, ", "), modValue, joinInts(keptRolls, ", "))
			}
		}
	}

	// Sum the kept rolls
	total := 0
	for _, roll := range keptRolls {
		total += roll
	}

	return total, breakdown, rolls, nil
}

// joinInts joins a slice of ints with a separator
func joinInts(nums []int, sep string) string {
	strs := make([]string, len(nums))
	for i, n := range nums {
		strs[i] = strconv.Itoa(n)
	}
	return strings.Join(strs, sep)
}

// --- Expression Definition Types ---

// ExpressionType defines different ways to generate a stat value
type ExpressionType string

const (
	ExprRoll     ExpressionType = "roll"     // Standard dice roll
	ExprBest     ExpressionType = "best"     // Best of N rolls
	ExprWorst    ExpressionType = "worst"    // Worst of N rolls
	ExprFixed    ExpressionType = "fixed"    // Fixed value
	ExprPointBuy ExpressionType = "pointbuy" // Point buy system
)

// StatExpression defines how to generate a stat value
type StatExpression struct {
	Type       ExpressionType `json:"type"`                 // Type of expression
	Expression string         `json:"expression,omitempty"` // Dice expression (for roll/best/worst)
	Count      int            `json:"count,omitempty"`      // Number of times to roll (for best/worst)
	Value      int            `json:"value,omitempty"`      // Fixed value (for fixed type)
	Min        int            `json:"min,omitempty"`        // Minimum allowed value
	Max        int            `json:"max,omitempty"`        // Maximum allowed value
}

// Evaluate evaluates the stat expression using the given roller
func (se *StatExpression) Evaluate(roller *Roller) (*RollResult, error) {
	var result *RollResult
	var err error

	switch se.Type {
	case ExprRoll:
		result, err = roller.Roll(se.Expression)
	case ExprBest:
		count := se.Count
		if count <= 0 {
			count = 2
		}
		result, err = roller.Best(se.Expression, count)
	case ExprWorst:
		count := se.Count
		if count <= 0 {
			count = 2
		}
		result, err = roller.Worst(se.Expression, count)
	case ExprFixed:
		result = &RollResult{
			Total:      se.Value,
			Expression: fmt.Sprintf("fixed(%d)", se.Value),
			Breakdown:  strconv.Itoa(se.Value),
		}
	case ExprPointBuy:
		// Point buy returns a base value; actual allocation happens elsewhere
		result = &RollResult{
			Total:      se.Value,
			Expression: "pointbuy",
			Breakdown:  fmt.Sprintf("base %d", se.Value),
		}
	default:
		// Default to roll if type not specified
		if se.Expression != "" {
			result, err = roller.Roll(se.Expression)
		} else {
			return nil, fmt.Errorf("unknown expression type: %s", se.Type)
		}
	}

	if err != nil {
		return nil, err
	}

	// Apply min/max constraints
	if se.Min > 0 && result.Total < se.Min {
		result.Total = se.Min
		result.Breakdown += fmt.Sprintf(" (min %d)", se.Min)
	}
	if se.Max > 0 && result.Total > se.Max {
		result.Total = se.Max
		result.Breakdown += fmt.Sprintf(" (max %d)", se.Max)
	}

	return result, nil
}
