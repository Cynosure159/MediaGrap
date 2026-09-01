package nfo

import (
	"encoding/xml"
	"strings"
)

func writeXML(value any) ([]byte, error) {
	body, err := xml.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(body, '\n')...), nil
}

func cleaned(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}
