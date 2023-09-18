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

func NewApplication(name, token string) *Application {
	return &Application{
		Name:  name,
		Token: token,
	}
}

func NewChat(applicationToken string, chatNum int) *Chat {
	return &Chat{
		Application: applicationToken,
		Number:      chatNum,
	}
}
