package types

type ChatMessage struct {
	Application   string `json:"application"`
	ChatNumber    int    `json:"chatNumber"`
	Body          string `json:"body"`
	MessageNumber int    `json:"messageNumber"`
}

type Chat struct {
	Application  string `json:"application"`
	Number       int    `json:"number"`
	MessageCount int    `json:"messageCount"`
}

type Application struct {
	Name      string `json:"name"`
	Token     string `json:"token"`
	ChatCount int    `json:"chatCount"`
}
