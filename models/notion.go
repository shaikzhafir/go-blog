package models

// this is for displaying on html
type HTMLBlock struct {
	Content string
}

// generic block type
type Block struct {
	Object string `json:"object"`
	ID     string `json:"id"`
	Parent struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"parent"`

	CreatedTime    string `json:"created_time"`
	LastEditedTime string `json:"last_edited_time"`
	HasChildren    bool   `json:"has_children"`
	Type           string `json:"type"`

	RawJSON map[string]interface{} `json:"-"`
	Content string                 `json:"content"`
}

// paragraph block type
type Paragraph struct {
	Block
	Paragraph struct {
		RichText []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    struct {
					URL string `json:"url"`
				}
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
	} `json:"paragraph"`
}

// heading 1 block type
type Heading1 struct {
	Block
	Heading1 struct {
		Text []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    string `json:"link"`
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
	} `json:"heading_1"`
}

// heading 2 block type
type Heading2 struct {
	Block
	Heading2 struct {
		Text []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    string `json:"link"`
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
	} `json:"heading_2"`
}

// heading 3 block type
type Heading3 struct {
	Block
	Heading3 struct {
		Text []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    string `json:"link"`
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
	} `json:"heading_3"`
}

type BulletedListItem struct {
	Block
	BulletedListItem struct {
		Text []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    string `json:"link"`
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
	} `json:"bulleted_list_item"`
}

type Page struct {
	Object         string `json:"object"`
	ID             string `json:"id"`
	CreatedTime    string `json:"created_time"`
	LastEditedTime string `json:"last_edited_time"`
	Parent         struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"parent"`
	Archived   bool `json:"archived"`
	Properties struct {
		Title []struct {
			ID    string `json:"id"`
			Type  string `json:"type"`
			Title []struct {
				Type string `json:"type"`
				Text struct {
					Content string `json:"content"`
					Link    string `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Strikethrough bool   `json:"strikethrough"`
					Underline     bool   `json:"underline"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string `json:"plain_text"`
				Href      string `json:"href"`
			} `json:"title"`
		} `json:"title"`
	} `json:"properties"`
}

type Code struct {
	Block
	Code struct {
		Caption []struct {
			Type string `json:"type"`
			Text []struct {
				Type string `json:"type"`
				Text struct {
					Content string `json:"content"`
					Link    string `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Strikethrough bool   `json:"strikethrough"`
					Underline     bool   `json:"underline"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string `json:"plain_text"`
				Href      string `json:"href"`
			} `json:"text"`
		} `json:"caption"`
		RichText []struct {
			Type string `json:"type"`
			Text struct {
				Content string `json:"content"`
				Link    string `json:"link"`
			} `json:"text"`
			Annotations struct {
				Bold          bool   `json:"bold"`
				Italic        bool   `json:"italic"`
				Strikethrough bool   `json:"strikethrough"`
				Underline     bool   `json:"underline"`
				Code          bool   `json:"code"`
				Color         string `json:"color"`
			} `json:"annotations"`
			PlainText string `json:"plain_text"`
			Href      string `json:"href"`
		} `json:"rich_text"`
		Language string `json:"language"`
	} `json:"code"`
}

type Image struct {
	Block
	Image struct {
		Caption []struct {
			Type string `json:"type"`
			Text []struct {
				Type string `json:"type"`
				Text struct {
					Content string `json:"content"`
					Link    string `json:"link"`
				} `json:"text"`
				Annotations struct {
					Bold          bool   `json:"bold"`
					Italic        bool   `json:"italic"`
					Strikethrough bool   `json:"strikethrough"`
					Underline     bool   `json:"underline"`
					Code          bool   `json:"code"`
					Color         string `json:"color"`
				} `json:"annotations"`
				PlainText string `json:"plain_text"`
				Href      string `json:"href"`
			} `json:"text"`
		} `json:"caption"`
		File struct {
			URL string `json:"url"`
		} `json:"file"`
	} `json:"image"`
}

type ReadingNowBlock struct {
	Title    string `json:"title"`
	ImageURL string `json:"image_url"`
	Author   string `json:"author"`
	Progress string `json:"progress"`
}
