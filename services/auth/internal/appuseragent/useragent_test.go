package appuseragent_test

import (
	"app/auth-service/internal/appuseragent"
	"testing"
)

type CaseUserAgentTest struct {
	name     string
	oldUA    string
	newUA    string
	expected bool
}

func TestValidateUserAgentSuccess(t *testing.T) {
	positiveTests := []CaseUserAgentTest{
		{
			name:     "Success - Identical Desktop Chrome Headers",
			oldUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			newUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: true,
		},
		{
			name:     "Success - Minor Browser Version Update Allowed",
			oldUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			newUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.6422.112 Safari/537.36",
			expected: true,
		},
		{
			name:     "Success - Identical API Client Bruno Headers",
			oldUA:    "bruno-runtime/3.5.3",
			newUA:    "bruno-runtime/3.5.3",
			expected: true,
		},
	}
	for _, test := range positiveTests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := appuseragent.ValidateUserAgent(test.oldUA, test.newUA)
			if result != test.expected {
				t.Errorf("ValidateUserAgent() failed for scenario '%s'. Got: %v, Expected: %v", test.name, result, test.expected)
			}
		})
	}
}
func TestValidateUserAgentNegative(t *testing.T) {
	negativeTests := []CaseUserAgentTest{
		{
			name:     "Failure - Critical Browser Family Hijack (Chrome to Firefox)",
			oldUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			newUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
			expected: false,
		},
		{
			name:     "Failure - Critical Device Type Hijack (Desktop to Mobile)",
			oldUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			newUA:    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			expected: false,
		},
		{
			name:     "Failure - Critical OS Hijack (Linux to Windows)",
			oldUA:    "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",
			newUA:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:120.0) Gecko/20100101 Firefox/120.0",
			expected: false,
		},
		{
			name:     "Failure - API Client Session Stolen by Real Browser (Bruno to Firefox)",
			oldUA:    "bruno-runtime/3.5.3",
			newUA:    "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:152.0) Gecko/20100101 Firefox/152.0",
			expected: false,
		},
		{
			name:     "Failure - Mobile Device Hijack (iPhone to Android)",
			oldUA:    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
			newUA:    "Mozilla/5.0 (Linux; Android 14; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36",
			expected: false,
		},
	}
	for _, test := range negativeTests {
		t.Run(test.name, func(t *testing.T) {
			result, _ := appuseragent.ValidateUserAgent(test.oldUA, test.newUA)
			if result != test.expected {
				t.Errorf("ValidateUserAgent() failed for scenario '%s'. Got: %v, Expected: %v", test.name, result, test.expected)
			}
		})
	}
}
