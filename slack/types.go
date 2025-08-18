package slack

// Block represents a Slack Block Kit block
type Block struct {
	Type     string         `json:"type"`
	Text     *TextObject    `json:"text,omitempty"`
	BlockID  string         `json:"block_id,omitempty"`
	Elements []BlockElement `json:"elements,omitempty"`
	Fields   []TextObject   `json:"fields,omitempty"`
}

// TextObject represents text in Slack blocks
type TextObject struct {
	Type  string `json:"type"` // "plain_text" or "mrkdwn"
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// BlockElement represents an element within a block
type BlockElement struct {
	Type     string      `json:"type"`
	Text     *TextObject `json:"text,omitempty"`
	Value    string      `json:"value,omitempty"`
	ActionID string      `json:"action_id,omitempty"`
	URL      string      `json:"url,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color      string       `json:"color,omitempty"`
	Fallback   string       `json:"fallback,omitempty"`
	Title      string       `json:"title,omitempty"`
	TitleLink  string       `json:"title_link,omitempty"`
	Text       string       `json:"text,omitempty"`
	Pretext    string       `json:"pretext,omitempty"`
	Footer     string       `json:"footer,omitempty"`
	FooterIcon string       `json:"footer_icon,omitempty"`
	Timestamp  int64        `json:"ts,omitempty"`
	Fields     []Field      `json:"fields,omitempty"`
	MrkdwnIn   []string     `json:"mrkdwn_in,omitempty"`
	Blocks     []Block      `json:"blocks,omitempty"`
}

// Field represents a field in an attachment
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// RichMessage represents a rich Slack message with blocks and attachments
type RichMessage struct {
	Text        string       `json:"text,omitempty"`
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// BatchResult represents the result of a batch send operation
type BatchResult struct {
	Index    int
	Response string
	Error    error
}

// Helper functions for creating common blocks

// NewSectionBlock creates a new section block
func NewSectionBlock(text string, markdown bool) Block {
	textType := "plain_text"
	if markdown {
		textType = "mrkdwn"
	}
	return Block{
		Type: "section",
		Text: &TextObject{
			Type: textType,
			Text: text,
		},
	}
}

// NewHeaderBlock creates a new header block
func NewHeaderBlock(text string) Block {
	return Block{
		Type: "header",
		Text: &TextObject{
			Type: "plain_text",
			Text: text,
		},
	}
}

// NewDividerBlock creates a new divider block
func NewDividerBlock() Block {
	return Block{
		Type: "divider",
	}
}

// NewContextBlock creates a new context block
func NewContextBlock(elements []BlockElement) Block {
	return Block{
		Type:     "context",
		Elements: elements,
	}
}

// NewButtonElement creates a new button element
func NewButtonElement(text, actionID, value string) BlockElement {
	return BlockElement{
		Type: "button",
		Text: &TextObject{
			Type: "plain_text",
			Text: text,
		},
		ActionID: actionID,
		Value:    value,
	}
}

// NewLinkButtonElement creates a new link button element
func NewLinkButtonElement(text, url string) BlockElement {
	return BlockElement{
		Type: "button",
		Text: &TextObject{
			Type: "plain_text",
			Text: text,
		},
		URL: url,
	}
}