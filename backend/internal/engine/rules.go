package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// EvaluateCondition tests whether a single condition matches the evaluation context.
// Returns false for absent attributes (condition does not match).
func EvaluateCondition(cond Condition, ctx EvaluationContext) bool {
	val, ok := resolveAttribute(cond.Attribute, ctx)
	if !ok {
		return false
	}

	switch cond.Operator {
	case OpEquals:
		return equals(val, cond.Value)
	case OpNotEquals:
		return !equals(val, cond.Value)
	case OpIn:
		return in(val, cond.Value)
	case OpNotIn:
		return !in(val, cond.Value)
	case OpContains:
		return stringOp(val, cond.Value, strings.Contains)
	case OpStartsWith:
		return stringOp(val, cond.Value, strings.HasPrefix)
	case OpEndsWith:
		return stringOp(val, cond.Value, strings.HasSuffix)
	case OpGreaterThan:
		return numericCmp(val, cond.Value, func(a, b float64) bool { return a > b })
	case OpLessThan:
		return numericCmp(val, cond.Value, func(a, b float64) bool { return a < b })
	case OpRegex:
		return regexMatch(val, cond.Value)
	case OpSemverMatch:
		return semverMatch(val, cond.Value)
	default:
		return false
	}
}

// AllConditionsMatch returns true when every condition in the slice matches.
func AllConditionsMatch(conditions []Condition, ctx EvaluationContext) bool {
	for _, c := range conditions {
		if !EvaluateCondition(c, ctx) {
			return false
		}
	}
	return true
}

// resolveAttribute looks up a dot-path attribute in the evaluation context.
// Supported paths: "subjectId", "environment", "tenantId", "attributes.<key>".
func resolveAttribute(path string, ctx EvaluationContext) (any, bool) {
	switch path {
	case "subjectId":
		return ctx.SubjectID, ctx.SubjectID != ""
	case "environment":
		return ctx.Environment, ctx.Environment != ""
	case "tenantId":
		return ctx.TenantID, ctx.TenantID != ""
	default:
		if strings.HasPrefix(path, "attributes.") {
			key := strings.TrimPrefix(path, "attributes.")
			if ctx.Attributes == nil {
				return nil, false
			}
			v, ok := ctx.Attributes[key]
			return v, ok
		}
		return nil, false
	}
}

func equals(actual, expected any) bool {
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

func in(actual, list any) bool {
	slice, ok := toSlice(list)
	if !ok {
		return false
	}
	actualStr := fmt.Sprintf("%v", actual)
	for _, item := range slice {
		if fmt.Sprintf("%v", item) == actualStr {
			return true
		}
	}
	return false
}

func toSlice(v any) ([]any, bool) {
	switch s := v.(type) {
	case []any:
		return s, true
	case []string:
		out := make([]any, len(s))
		for i, item := range s {
			out[i] = item
		}
		return out, true
	default:
		return nil, false
	}
}

func stringOp(actual, expected any, fn func(string, string) bool) bool {
	a, ok := toString(actual)
	if !ok {
		return false
	}
	b, ok := toString(expected)
	if !ok {
		return false
	}
	return fn(a, b)
}

func toString(v any) (string, bool) {
	switch s := v.(type) {
	case string:
		return s, true
	default:
		return fmt.Sprintf("%v", v), true
	}
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

func numericCmp(actual, expected any, cmp func(float64, float64) bool) bool {
	a, ok := toFloat64(actual)
	if !ok {
		return false
	}
	b, ok := toFloat64(expected)
	if !ok {
		return false
	}
	return cmp(a, b)
}

func regexMatch(actual, pattern any) bool {
	a, ok := toString(actual)
	if !ok {
		return false
	}
	p, ok := toString(pattern)
	if !ok {
		return false
	}
	re, err := regexp.Compile(p)
	if err != nil {
		return false
	}
	return re.MatchString(a)
}

// semverMatch checks if an actual version satisfies a constraint.
// Supported constraint prefixes: >=, <=, >, <, = (or exact match).
func semverMatch(actual, constraint any) bool {
	ver, ok := toString(actual)
	if !ok {
		return false
	}
	cstr, ok := toString(constraint)
	if !ok {
		return false
	}

	ver = strings.TrimPrefix(ver, "v")

	var op string
	var target string
	for _, prefix := range []string{">=", "<=", ">", "<", "="} {
		if strings.HasPrefix(cstr, prefix) {
			op = prefix
			target = strings.TrimSpace(strings.TrimPrefix(cstr, prefix))
			break
		}
	}
	if op == "" {
		op = "="
		target = cstr
	}
	target = strings.TrimPrefix(target, "v")

	cmpResult := compareSemver(ver, target)

	switch op {
	case ">=":
		return cmpResult >= 0
	case "<=":
		return cmpResult <= 0
	case ">":
		return cmpResult > 0
	case "<":
		return cmpResult < 0
	case "=":
		return cmpResult == 0
	default:
		return false
	}
}

// compareSemver returns -1, 0, or 1 comparing two semver strings.
// Only compares major.minor.patch numerically; ignores pre-release/build metadata.
func compareSemver(a, b string) int {
	aParts := parseSemverParts(a)
	bParts := parseSemverParts(b)
	for i := 0; i < 3; i++ {
		if aParts[i] < bParts[i] {
			return -1
		}
		if aParts[i] > bParts[i] {
			return 1
		}
	}
	return 0
}

func parseSemverParts(v string) [3]int {
	// Strip pre-release/build metadata
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i, p := range parts {
		if i >= 3 {
			break
		}
		n := 0
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				n = n*10 + int(ch-'0')
			}
		}
		result[i] = n
	}
	return result
}
