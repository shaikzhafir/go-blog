package notion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	log "htmx-blog/logging"
	"htmx-blog/models"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type notionClient struct {
	NotionToken string
	DatabaseID  string
	Converter   Converter
}

type Entry struct {
	Object         string     `json:"object"`
	ID             string     `json:"id"`
	CreatedTime    string     `json:"created_time"`
	LastEditedTime string     `json:"last_edited_time"`
	Properties     Properties `json:"properties"`
}

type SlugEntry struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CreatedTime string `json:"created_time"`
	Slug        string `json:"slug"`
}

type Properties struct {
	Slug Slug `json:"slug"`
	Name Name `json:"Name"`
}

type Name struct {
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
}

type Slug struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
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
}

type QueryDBResponse struct {
	Object     string  `json:"object"`
	Results    []Entry `json:"results"`
	NextCursor string  `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}

type QueryBlockChildrenResponse struct {
	Object string `json:"object"`
	// its only rawmessage because we need to iterate
	Results    []json.RawMessage `json:"results"`
	Type       string            `json:"type"`
	NextCursor string            `json:"next_cursor"`
	HasMore    bool              `json:"has_more"`
}

type NotionClient interface {
	GetBlockChildren(blockID string) ([]json.RawMessage, error)
	GetBlock(blockID string) (models.Block, error)
	GetPage(pageID string) (models.Page, error)
	GetAllPosts(databaseID string) (map[string]string, error)
	GetSlugEntries(databaseID string) ([]SlugEntry, error)
	GetDatabaseID() string
	ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte) error
}

// this should only be called by redis service to get the data
// from notion and store it in redis
// for now , no redis
func NewNotionClient() NotionClient {
	notionToken := os.Getenv("NOTION_TOKEN")
	if notionToken == "" {
		panic("NOTION_TOKEN not set")
	}
	databaseID := os.Getenv("NOTION_DATABASE_ID")
	if databaseID == "" {
		panic("NOTION_DATABASE_ID not set")
	}
	return &notionClient{
		NotionToken: notionToken,
		DatabaseID:  databaseID,
	}
}

// GetBlock implements NotionClient.
func (nc *notionClient) GetBlock(blockID string) (models.Block, error) {
	panic("unimplemented")
}

// GetBlockChildren implements NotionClient.
func (nc *notionClient) GetBlockChildren(blockID string) ([]json.RawMessage, error) {
	var body []byte
	br := bytes.NewBuffer(body)
	req, err := http.NewRequest("GET", "https://api.notion.com/v1/blocks/"+blockID+"/children", br)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	//check for rate limit
	// todo add retry logic
	if resp.StatusCode == http.StatusTooManyRequests {
		log.Info("rate limit hit u greedy fucker!")
		return nil, err
	}

	var response QueryBlockChildrenResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return response.Results, nil
}

// GetPage implements NotionClient.
func (nc *notionClient) GetPage(pageID string) (models.Page, error) {
	var body []byte
	br := bytes.NewBuffer(body)
	req, err := http.NewRequest("GET", "https://api.notion.com/v1/pages/"+pageID, br)
	if err != nil {
		return models.Page{}, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.Page{}, err
	}
	defer resp.Body.Close()
	var page models.Page
	err = json.NewDecoder(resp.Body).Decode(&page)
	if err != nil {
		return models.Page{}, err
	}
	return page, nil
}

func (nc *notionClient) GetAllPosts(databaseID string) (map[string]string, error) {
	var body []byte
	br := bytes.NewBuffer(body)
	req, err := http.NewRequest("POST", "https://api.notion.com/v1/databases/"+databaseID+"/query", br)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var dbResponse QueryDBResponse
	err = json.NewDecoder(resp.Body).Decode(&dbResponse)
	if err != nil {
		return nil, err
	}

	posts := make(map[string]string)

	for _, entry := range dbResponse.Results {
		// an empty RichText is not nil but an empty slice
		if entry.Properties.Slug.RichText == nil || len(entry.Properties.Slug.RichText) == 0 {
			continue
		}
		if entry.Properties.Slug.RichText[0].PlainText == "" {
			continue
		}
		posts[entry.ID] = entry.Properties.Slug.RichText[0].PlainText
	}

	return posts, nil
}

func (nc *notionClient) GetSlugEntries(databaseID string) ([]SlugEntry, error) {
	var body []byte
	br := bytes.NewBuffer(body)
	req, err := http.NewRequest("POST", "https://api.notion.com/v1/databases/"+databaseID+"/query", br)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2022-06-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var dbResponse QueryDBResponse
	err = json.NewDecoder(resp.Body).Decode(&dbResponse)
	if err != nil {
		return nil, err
	}

	slugEntries := []SlugEntry{}

	for _, entry := range dbResponse.Results {
		// an empty RichText is not nil but an empty slice
		if entry.Properties.Slug.RichText == nil || len(entry.Properties.Slug.RichText) == 0 || len(entry.Properties.Name.Title) == 0 {
			continue
		}
		if entry.Properties.Slug.RichText[0].PlainText == "" {
			continue
		}

		slugEntry := SlugEntry{
			ID:          entry.ID,
			Title:       entry.Properties.Name.Title[0].PlainText,
			CreatedTime: entry.CreatedTime,
			Slug:        entry.Properties.Slug.RichText[0].PlainText,
		}

		// append to slice
		slugEntries = append(slugEntries, slugEntry)
	}

	return slugEntries, nil
}

func (nc *notionClient) GetDatabaseID() string {
	return nc.DatabaseID
}

type converter struct {
	rawBlock []byte
	writer   io.Writer
}

// RenderBulletedListItem implements Converter.
func (c *converter) RenderBulletedListItem() error {
	// first unmarshal into heading1 block
	var block models.BulletedListItem
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/bulleted_list_item.html")
	if err != nil {
		log.Error("error getting absolute path: %v", err)
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.BulletedListItem.Text) == 0 {
		return nil
	}
	block.Content = block.BulletedListItem.Text[0].Text.Content
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

// RenderChildPage implements Converter.
func (*converter) RenderChildPage() error {
	return nil
}

// RenderHeading1 implements Converter.
func (c *converter) RenderHeading1() error {
	// first unmarshal into heading1 block
	var block models.Heading1
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/heading_1.html")
	if err != nil {
		log.Error("error getting absolute path: %v", err)
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.Heading1.Text) == 0 {
		return nil
	}
	block.Content = block.Heading1.Text[0].Text.Content
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

// RenderHeading2 implements Converter.
func (c *converter) RenderHeading2() error {
	// first unmarshal into heading1 block
	var block models.Heading2
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/heading_2.html")
	if err != nil {
		log.Error("error getting absolute path: %v", err)
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.Heading2.Text) == 0 {
		return nil
	}
	block.Content = block.Heading2.Text[0].Text.Content
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

// RenderHeading3 implements Converter.
func (c *converter) RenderHeading3() error {
	// first unmarshal into heading1 block
	var block models.Heading3
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/heading_3.html")
	if err != nil {
		log.Error("error getting absolute path: %v", err)
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.Heading3.Text) == 0 {
		return nil
	}
	block.Content = block.Heading3.Text[0].Text.Content
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

// RenderNumberedListItem implements Converter.
func (*converter) RenderNumberedListItem() error {
	return nil
}

// RenderParagraph will unmarshal raw JSON block into a paragraph block
// it will then execute the paragraph template with content based on the paragraph block
func (c *converter) RenderParagraph() error {
	// first unmarshal into paragraph block
	var block models.Paragraph
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	if len(block.Paragraph.RichText) == 0 {
		return nil
	}
	// for loop in case rich_text has more than one element
	for _, richText := range block.Paragraph.RichText {
		htmlBlock := models.HTMLBlock{}
		// load template
		// if it has link, use link template
		templatePath, err := filepath.Abs("./templates/notionBlocks/paragraph.html")
		if err != nil {
			return err
		}
		if richText.Text.Link.URL != "" {
			templatePath, err = filepath.Abs("./templates/notionBlocks/link.html")
			if err != nil {
				return err
			}
		}
		tmpl, err := template.ParseFiles(templatePath)
		if err != nil {
			return err
		}

		htmlBlock.Content = richText.Text.Content

		if richText.Text.Link.URL != "" {
			htmlBlock.Content = richText.Text.Link.URL
		}
		err = tmpl.Execute(c.writer, htmlBlock)
		if err != nil {
			return err
		}
		continue
	}
	return nil
}

// RenderToDoItem implements Converter.
func (*converter) RenderToDoItem() error {
	return nil
}

// RenderToggle implements Converter.
func (*converter) RenderToggle() error {
	return nil
}

// RenderUnsupported implements Converter.
func (*converter) RenderUnsupported() error {
	return nil
}

func (c *converter) RenderImage() error {
	// first unmarshal into paragraph block
	var block models.Image
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/image.html")
	if err != nil {
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.Image.File.URL) == 0 {
		return nil
	}
	block.Content = block.Image.File.URL
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

func (c *converter) RenderCode() error {
	// first unmarshal into paragraph block
	var block models.Code
	err := json.Unmarshal(c.rawBlock, &block)
	if err != nil {
		return err
	}
	// load template
	templatePath, err := filepath.Abs("./templates/notionBlocks/code.html")
	if err != nil {
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	if len(block.Code.RichText) == 0 {
		return nil
	}
	block.Content = block.Code.RichText[0].Text.Content
	err = tmpl.Execute(c.writer, block)
	if err != nil {
		return err
	}
	return nil
}

func NewConverter(writer io.Writer, rawBlock []byte) Converter {
	return &converter{
		rawBlock: rawBlock,
		writer:   writer,
	}
}

type Converter interface {
	RenderParagraph() error
	RenderHeading1() error
	RenderHeading2() error
	RenderHeading3() error
	RenderBulletedListItem() error
	RenderNumberedListItem() error
	RenderCode() error
	RenderToDoItem() error
	RenderToggle() error
	RenderChildPage() error
	RenderUnsupported() error
	RenderImage() error
}

func (nc *notionClient) ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte) error {
	// unmarshal to find block type
	var b models.Block
	err := json.Unmarshal(rawBlock, &b)
	if err != nil {
		return err
	}

	c := NewConverter(writer, rawBlock)

	switch b.Type {
	case "paragraph":
		return c.RenderParagraph()
	case "heading_1":
		return c.RenderHeading1()
	case "heading_2":
		return c.RenderHeading2()
	case "heading_3":
		return c.RenderHeading3()
	case "bulleted_list_item":
		return c.RenderBulletedListItem()
	case "image":
		return c.RenderImage()
	case "code":
		return c.RenderCode()
	default:
		return nil
	}
}

// StoreNotionImage stores the image locally and updates the rawBlock with the new image url
// first it will get the existing fresh image url from notion aws image url
// then it will download the image from the aws image url
// then it will store the image locally
func StoreNotionImage(rawBlocks []json.RawMessage, i int) error {
	var imageBlock models.Image
	err := json.Unmarshal(rawBlocks[i], &imageBlock)
	if err != nil {
		log.Error("error unmarshalling imageblock: %v", err)
	}
	awsImageURL := imageBlock.Image.File.URL
	// read and write image to r2, then update the rawBlock with the new image url
	// Download file from S3
	resp, err := http.Get(awsImageURL)
	if err != nil {
		return fmt.Errorf("error downloading image from s3: %v", err)
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading image bytes: %v", err)
	}
	// store locally in ./images
	// Ensure the folder exists
	absPath, err := filepath.Abs("./images")
	if err != nil {
		return fmt.Errorf("error getting absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		os.Mkdir(absPath, os.ModePerm)
	}

	filePath := filepath.Join(absPath, "/", imageBlock.ID+".png")
	err = os.WriteFile(filePath, imageBytes, 0755)
	if err != nil {
		return fmt.Errorf("error writing image to file: %v", err)
	}

	imageBlock.Image.File.URL = "https://cloud.shaikzhafir.com/images/" + imageBlock.ID + ".png"
	if os.Getenv("DEV") == "true" {
		imageBlock.Image.File.URL = "/images/" + imageBlock.ID + ".png"
	}

	// update rawBlock with new image url
	rawBlocks[i], err = json.Marshal(imageBlock)
	if err != nil {
		return fmt.Errorf("error marshalling imageblock: %v", err)
	}
	return nil
}
