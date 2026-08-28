package tools

func GetTools() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "take_screenshot",
				"description": "Captures the desktop screen as a base64 JPEG.",
				"parameters":  map[string]interface{}{"type": "object", "properties": map[string]interface{}{}},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "click_mouse",
				"description": "Moves mouse to coordinates (x, y) and clicks.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"x":      map[string]interface{}{"type": "integer"},
						"y":      map[string]interface{}{"type": "integer"},
						"button": map[string]interface{}{"type": "string", "enum": []string{"left", "right", "center"}},
						"double": map[string]interface{}{"type": "boolean"},
					},
					"required": []string{"x", "y", "button"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "type_text",
				"description": "Simulates typing text directly into the active window.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"text": map[string]interface{}{"type": "string"},
					},
					"required": []string{"text"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "press_key_combination",
				"description": "Presses keyboard shortcuts like Ctrl+C, Win+R, or Enter.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key":       map[string]interface{}{"type": "string"},
						"modifiers": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					},
					"required": []string{"key"},
				},
			},
		},
		{
			"type": "function",
			"function": map[string]interface{}{
				"name":        "run_powershell",
				"description": "Executes shell commands or scripts in PowerShell 7.",
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"command": map[string]interface{}{"type": "string"},
					},
					"required": []string{"command"},
				},
			},
		},
	}
}