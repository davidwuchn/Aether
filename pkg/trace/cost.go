package trace

// modelRate holds the per-1K-token pricing for a model.
type modelRate struct {
	InputRate  float64
	OutputRate float64
}

// modelRates maps model names to USD rates per 1,000 tokens.
// Rates are approximate and should be updated as providers change pricing.
var modelRates = map[string]modelRate{
	"claude-sonnet-4-20250514": {InputRate: 3.0, OutputRate: 15.0},
	"claude-sonnet-4":          {InputRate: 3.0, OutputRate: 15.0},
	"claude-sonnet":            {InputRate: 3.0, OutputRate: 15.0},
	"claude-opus-4-20250514":   {InputRate: 15.0, OutputRate: 75.0},
	"claude-opus-4":            {InputRate: 15.0, OutputRate: 75.0},
	"claude-opus":              {InputRate: 15.0, OutputRate: 75.0},
	"claude-haiku-3-20240307":  {InputRate: 0.25, OutputRate: 1.25},
	"claude-haiku-3":           {InputRate: 0.25, OutputRate: 1.25},
	"claude-haiku":             {InputRate: 0.25, OutputRate: 1.25},
	"gpt-4":                    {InputRate: 30.0, OutputRate: 60.0},
	"gpt-4-turbo":              {InputRate: 10.0, OutputRate: 30.0},
	"gpt-4-turbo-preview":      {InputRate: 10.0, OutputRate: 30.0},
	"gpt-3.5-turbo":            {InputRate: 0.5, OutputRate: 1.5},
	"gpt-3.5-turbo-0125":       {InputRate: 0.5, OutputRate: 1.5},
}

// CalculateCost returns the estimated USD cost for an LLM call.
// It looks up the model in modelRates and returns 0 if unknown.
func CalculateCost(model string, inputTokens, outputTokens int64) float64 {
	rate, ok := modelRates[model]
	if !ok {
		return 0
	}
	inputCost := float64(inputTokens) * rate.InputRate / 1000.0
	outputCost := float64(outputTokens) * rate.OutputRate / 1000.0
	return inputCost + outputCost
}
