package sarif

type Log struct {
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

type Tool struct {
	Driver Driver `json:"driver"`
}

type Driver struct {
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

type Rule struct {
	ID               string `json:"id"`
	ShortDescription Text   `json:"shortDescription"`
	FullDescription  Text   `json:"fullDescription"`
	Help             Help   `json:"help"`
	HelpURI          string `json:"helpUri"`
	DefaultConfig    struct {
		Level string `json:"level"`
	} `json:"defaultConfiguration"`
}

type Help struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

type Text struct {
	Text string `json:"text"`
}

type Result struct {
	RuleID    string     `json:"ruleId"`
	RuleIndex *int       `json:"ruleIndex"`
	Level     string     `json:"level"`
	Message   Text       `json:"message"`
	Locations []Location `json:"locations"`
}

type Location struct {
	Physical struct {
		Artifact struct {
			URI string `json:"uri"`
		} `json:"artifactLocation"`
		Region struct {
			StartLine   int  `json:"startLine"`
			StartColumn int  `json:"startColumn"`
			EndLine     int  `json:"endLine"`
			EndColumn   int  `json:"endColumn"`
			Snippet     Text `json:"snippet"`
		} `json:"region"`
	} `json:"physicalLocation"`
}
