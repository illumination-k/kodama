package config

import "testing"

func TestCoalesceString(t *testing.T) {
	tests := []struct {
		name          string
		cliValue      string
		resolvedValue string
		expected      string
	}{
		{
			name:          "CLI value provided",
			cliValue:      "cli-value",
			resolvedValue: "resolved-value",
			expected:      "cli-value",
		},
		{
			name:          "CLI value empty",
			cliValue:      "",
			resolvedValue: "resolved-value",
			expected:      "resolved-value",
		},
		{
			name:          "both empty",
			cliValue:      "",
			resolvedValue: "",
			expected:      "",
		},
		{
			name:          "both provided",
			cliValue:      "cli",
			resolvedValue: "resolved",
			expected:      "cli",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoalesceString(tt.cliValue, tt.resolvedValue)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestCoalesceInt(t *testing.T) {
	tests := []struct {
		name          string
		cliValue      int
		resolvedValue int
		expected      int
	}{
		{
			name:          "CLI value provided",
			cliValue:      100,
			resolvedValue: 200,
			expected:      100,
		},
		{
			name:          "CLI value zero",
			cliValue:      0,
			resolvedValue: 200,
			expected:      200,
		},
		{
			name:          "both zero",
			cliValue:      0,
			resolvedValue: 0,
			expected:      0,
		},
		{
			name:          "both provided",
			cliValue:      50,
			resolvedValue: 100,
			expected:      50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoalesceInt(tt.cliValue, tt.resolvedValue)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestCoalesceBool(t *testing.T) {
	tests := []struct {
		name          string
		cliValue      bool
		resolvedValue bool
		condition     bool // whether CLI flag was explicitly set
		expected      bool
	}{
		{
			name:          "CLI explicitly set to true",
			cliValue:      true,
			resolvedValue: false,
			condition:     true,
			expected:      true,
		},
		{
			name:          "CLI explicitly set to false",
			cliValue:      false,
			resolvedValue: true,
			condition:     true,
			expected:      false,
		},
		{
			name:          "CLI not set, use resolved true",
			cliValue:      false,
			resolvedValue: true,
			condition:     false,
			expected:      true,
		},
		{
			name:          "CLI not set, use resolved false",
			cliValue:      false,
			resolvedValue: false,
			condition:     false,
			expected:      false,
		},
		{
			name:          "CLI set to true, resolved false",
			cliValue:      true,
			resolvedValue: false,
			condition:     true,
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoalesceBool(tt.cliValue, tt.resolvedValue, tt.condition)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCoalesceMap(t *testing.T) {
	tests := []struct {
		name         string
		cliMap       map[string]string
		resolvedMap  map[string]string
		expectedKeys map[string]string
	}{
		{
			name: "CLI overrides resolved",
			cliMap: map[string]string{
				"key1": "cli-value",
			},
			resolvedMap: map[string]string{
				"key1": "resolved-value",
				"key2": "resolved-value2",
			},
			expectedKeys: map[string]string{
				"key1": "cli-value",
				"key2": "resolved-value2",
			},
		},
		{
			name:   "CLI map nil",
			cliMap: nil,
			resolvedMap: map[string]string{
				"key1": "resolved-value",
			},
			expectedKeys: map[string]string{
				"key1": "resolved-value",
			},
		},
		{
			name:   "CLI map empty",
			cliMap: map[string]string{},
			resolvedMap: map[string]string{
				"key1": "resolved-value",
			},
			expectedKeys: map[string]string{
				"key1": "resolved-value",
			},
		},
		{
			name: "CLI adds new keys",
			cliMap: map[string]string{
				"key2": "cli-value2",
			},
			resolvedMap: map[string]string{
				"key1": "resolved-value1",
			},
			expectedKeys: map[string]string{
				"key1": "resolved-value1",
				"key2": "cli-value2",
			},
		},
		{
			name:         "both empty",
			cliMap:       map[string]string{},
			resolvedMap:  map[string]string{},
			expectedKeys: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CoalesceMap(tt.cliMap, tt.resolvedMap)

			// Check length
			if len(result) != len(tt.expectedKeys) {
				t.Errorf("expected map length %d, got %d", len(tt.expectedKeys), len(result))
			}

			// Check each key-value pair
			for k, expectedV := range tt.expectedKeys {
				if actualV, exists := result[k]; !exists {
					t.Errorf("expected key %q not found in result", k)
				} else if actualV != expectedV {
					t.Errorf("for key %q: expected value %q, got %q", k, expectedV, actualV)
				}
			}

			// Check for unexpected keys
			for k := range result {
				if _, exists := tt.expectedKeys[k]; !exists {
					t.Errorf("unexpected key %q in result", k)
				}
			}
		})
	}
}
