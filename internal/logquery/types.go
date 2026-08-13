package logquery

type Query struct {
	Namespace string
	Pod       string
	Container string
	From      string
	Keyword   string
	Level     string
	Limit     int64
	Previous  bool
}

type Line struct {
	Raw string `json:"raw"`
}

type Result struct {
	Source    string `json:"source"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container,omitempty"`
	Lines     []Line `json:"lines"`
	Total     int    `json:"total"`
}

type StreamMessage struct {
	Type    string `json:"type"`
	Line    *Line  `json:"line,omitempty"`
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}
