// Package renamepattern defines the deliberately small rename-template grammar.
package renamepattern

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

type Expression struct {
	Token    string
	Prefix   string
	Suffix   string
	Optional bool
}

type Part struct {
	Literal    string
	Expression *Expression
}

func Allowed(names ...string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		allowed[name] = struct{}{}
	}
	return allowed
}

func Parse(pattern string, allowed map[string]struct{}) ([]Part, error) {
	parts := make([]Part, 0, 8)
	literalStart := 0
	for i := 0; i < len(pattern); {
		if i+1 >= len(pattern) || pattern[i] != '$' || pattern[i+1] != '{' {
			i++
			continue
		}
		if i > literalStart {
			parts = append(parts, Part{Literal: pattern[literalStart:i]})
		}
		end := strings.IndexByte(pattern[i+2:], '}')
		if end < 0 {
			return nil, errors.New("malformed naming expression")
		}
		end += i + 2
		body := pattern[i+2 : end]
		if strings.Contains(body, "${") || strings.Contains(body, "}") {
			return nil, errors.New("nested naming expressions are not supported")
		}
		var expression Expression
		switch strings.Count(body, ",") {
		case 0:
			expression.Token = trimGoSpace(body)
		case 2:
			fields := strings.Split(body, ",")
			expression = Expression{Prefix: fields[0], Token: trimGoSpace(fields[1]), Suffix: fields[2], Optional: true}
			if err := validateAffix(expression.Prefix); err != nil {
				return nil, err
			}
			if err := validateAffix(expression.Suffix); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("naming expressions must contain either no commas or exactly two commas")
		}
		if len(expression.Token) == 0 || !validTokenName(expression.Token) {
			return nil, fmt.Errorf("malformed naming token %q", expression.Token)
		}
		if _, ok := allowed[expression.Token]; !ok {
			return nil, fmt.Errorf("unsupported naming token %q", expression.Token)
		}
		parts = append(parts, Part{Expression: &expression})
		i = end + 1
		literalStart = i
	}
	if literalStart < len(pattern) {
		parts = append(parts, Part{Literal: pattern[literalStart:]})
	}
	for _, part := range parts {
		if strings.ContainsAny(part.Literal, "${}") {
			return nil, errors.New("malformed naming expression")
		}
	}
	return parts, nil
}

func Render(pattern string, values map[string]string, allowed map[string]struct{}) (string, error) {
	parts, err := Parse(pattern, allowed)
	if err != nil {
		return "", err
	}
	var rendered strings.Builder
	for _, part := range parts {
		if part.Expression == nil {
			rendered.WriteString(part.Literal)
			continue
		}
		rawValue := values[part.Expression.Token]
		if strings.TrimSpace(rawValue) == "" {
			if part.Expression.Optional {
				continue
			}
			return "", fmt.Errorf("naming token %q has no available value", part.Expression.Token)
		}
		value := strings.NewReplacer("/", " ", "\\", " ").Replace(rawValue)
		if part.Expression.Optional {
			rendered.WriteString(part.Expression.Prefix)
		}
		rendered.WriteString(value)
		if part.Expression.Optional {
			rendered.WriteString(part.Expression.Suffix)
		}
	}
	return rendered.String(), nil
}

func Validate(pattern string, allowed map[string]struct{}) error {
	if pattern == "" || len(pattern) > 1024 {
		return errors.New("rename pattern must contain 1–1024 bytes")
	}
	if strings.ContainsAny(pattern, `\\:*?"<>|`) || strings.IndexFunc(pattern, unicode.IsControl) >= 0 || strings.HasPrefix(pattern, "/") {
		return errors.New("rename pattern must be a safe relative path")
	}
	if _, err := Parse(pattern, allowed); err != nil {
		return err
	}
	for _, segment := range strings.Split(pattern, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return errors.New("rename pattern contains an invalid path segment")
		}
	}
	return nil
}

func trimGoSpace(value string) string {
	return strings.TrimFunc(value, unicode.IsSpace)
}

func validTokenName(name string) bool {
	if name == "" || (name[0] < 'A' || name[0] > 'Z') && (name[0] < 'a' || name[0] > 'z') {
		return false
	}
	for _, char := range name[1:] {
		if (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && (char < '0' || char > '9') {
			return false
		}
	}
	return true
}

func validateAffix(affix string) error {
	if strings.ContainsAny(affix, `/\\`) || strings.IndexFunc(affix, unicode.IsControl) >= 0 || strings.ContainsAny(affix, `:*?"<>|{}`) {
		return errors.New("optional naming affixes must be safe filename text")
	}
	if affix == "." || affix == ".." {
		return errors.New("optional naming affixes cannot be traversal segments")
	}
	return nil
}
