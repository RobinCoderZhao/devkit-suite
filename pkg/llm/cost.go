package llm

// Token pricing per 1M tokens (USD) as of 2025.
var pricing = map[string]modelPrice{
	// OpenAI
	"gpt-4o":        {Input: 2.50, Output: 10.00},
	"gpt-4o-mini":   {Input: 0.15, Output: 0.60},
	"gpt-4-turbo":   {Input: 10.00, Output: 30.00},
	"gpt-3.5-turbo": {Input: 0.50, Output: 1.50},
	"o1":            {Input: 15.00, Output: 60.00},
	"o1-mini":       {Input: 3.00, Output: 12.00},

	// Gemini
	"gemini-2.0-flash":      {Input: 0.10, Output: 0.40},
	"gemini-2.0-flash-lite": {Input: 0.0, Output: 0.0},
	"gemini-1.5-pro":        {Input: 1.25, Output: 5.00},
	"gemini-1.5-flash":      {Input: 0.075, Output: 0.30},

	// Claude
	"claude-3-5-sonnet-20241022": {Input: 3.00, Output: 15.00},
	"claude-3-5-haiku-20241022":  {Input: 0.80, Output: 4.00},
	"claude-3-opus-20240229":     {Input: 15.00, Output: 75.00},

	// MiniMax
	"MiniMax-M2.5":           {Input: 1.10, Output: 4.40},
	"MiniMax-M2.5-highspeed": {Input: 0.55, Output: 2.20},
	"MiniMax-M2.1":           {Input: 0.80, Output: 3.20},
	"MiniMax-M2.1-highspeed": {Input: 0.40, Output: 1.60},
	"MiniMax-M2":             {Input: 0.50, Output: 2.00},
}

type modelPrice struct {
	Input  float64 // per 1M input tokens
	Output float64 // per 1M output tokens
}

// EstimateCost returns the estimated cost in USD for the given model and token counts.
func EstimateCost(model string, tokensIn, tokensOut int) float64 {
	p, ok := pricing[model]
	if !ok {
		return 0
	}
	return (float64(tokensIn) * p.Input / 1_000_000) + (float64(tokensOut) * p.Output / 1_000_000)
}
