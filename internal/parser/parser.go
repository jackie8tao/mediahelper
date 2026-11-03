package parser

// Parser defines the universal AIGC media title parser interface.
type Parser interface {
	// Name returns the parser client name, e.g. "OpenAI", "Gemini"
	Name() string
}
