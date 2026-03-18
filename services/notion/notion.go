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
	"strings"
	"time"
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

type ReadingNow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	CreatedTime string `json:"created_time"`
	Image       string `json:"image,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Progress    string `json:"progress,omitempty"`
	Author      string `json:"author,omitempty"`
}

type Properties struct {
	Slug     Slug           `json:"slug"`
	Name     Name           `json:"name"`
	Author   Slug           `json:"author"`
	Image    PropertyImage  `json:"image"`
	Comment  Slug           `json:"comment"`
	Progress PropertyNumber `json:"progress"`
}

type Name struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Title []struct {
		Type string `json:"type"`
		Text struct {
			Content string `json:"content"`
			Link    any    `json:"link"`
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
			Link    any    `json:"link"`
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

type PropertyImage struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Files []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		External struct {
			URL string `json:"url"`
		} `json:"external"`
		File struct {
			URL        string `json:"url"`
			ExpiryTime string `json:"expiry_time"`
		} `json:"file"`
	} `json:"files"`
}

type PropertyNumber struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Number *float64 `json:"number"`
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
	GetAllPosts(databaseID string, filter string) (map[string]string, error)
	GetSlugEntries(databaseID string, filter string) ([]SlugEntry, error)
	GetReadingNowEntries(datasourceID string, filter string) ([]ReadingNow, error)
	GetDatabaseID() string
	ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte, postType string) error
}

// this should only be called by cache service to get the data
// from notion and store it in JSON file cache
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

func (nc *notionClient) GetAllPosts(databaseID string, filter string) (map[string]string, error) {
	bodyPayload := bytes.NewBuffer([]byte(fmt.Sprintf(`{
		"filter": {
		"property": "tags",
		"multi_select": {
			"contains": "%s"
		}
	}
	}`, filter)))
	req, err := http.NewRequest("POST", "https://api.notion.com/v1/databases/"+databaseID+"/query", bodyPayload)
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

func (nc *notionClient) GetSlugEntries(datasourceID string, filter string) ([]SlugEntry, error) {
	bodyPayload := bytes.NewBuffer([]byte(fmt.Sprintf(`{
		"filter": {
			"property": "tags",
			"multi_select": {
				"contains": "%s"
			}
		},
		"sorts": [
			{
				"timestamp": "created_time",
				"direction": "descending"
			}
		]
	}`, filter)))
	req, err := http.NewRequest("POST", "https://api.notion.com/v1/data_sources/"+datasourceID+"/query", bodyPayload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2025-09-03")
	log.Info("making request to notion for slug entries, %+v", req)
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

		parsedTime, err := time.Parse(time.RFC3339, entry.CreatedTime)
		if err != nil {
			log.Error("error parsing time: %v", err)
		} else {
			readableFormat := "January 2, 2006 at 15:04"
			entry.CreatedTime = parsedTime.Format(readableFormat)
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

func (nc *notionClient) GetReadingNowEntries(datasourceID string, filter string) ([]ReadingNow, error) {
	bodyPayload := bytes.NewBuffer([]byte(fmt.Sprintf(`{
		"filter": {
			"property": "tags",
			"multi_select": {
				"contains": "%s"
			}
		},
		"sorts": [
			{
				"timestamp": "created_time",
				"direction": "descending"
			}
		]
	}`, filter)))
	req, err := http.NewRequest("POST", "https://api.notion.com/v1/data_sources/"+datasourceID+"/query", bodyPayload)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+nc.NotionToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", "2025-09-03")
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

	readnowEntries := []ReadingNow{}
	for _, entry := range dbResponse.Results {
		// Check if title exists
		if len(entry.Properties.Name.Title) == 0 {
			continue
		}

		parsedTime, err := time.Parse(time.RFC3339, entry.CreatedTime)
		if err != nil {
			log.Error("error parsing time: %v", err)
		} else {
			readableFormat := "January 2, 2006 at 15:04"
			entry.CreatedTime = parsedTime.Format(readableFormat)
		}

		slugEntry := ReadingNow{
			ID:          entry.ID,
			Title:       entry.Properties.Name.Title[0].PlainText,
			CreatedTime: entry.CreatedTime,
		}

		// Handle author (rich_text field)
		if len(entry.Properties.Author.RichText) > 0 {
			slugEntry.Author = entry.Properties.Author.RichText[0].PlainText
		}
		// Handle progress (number field, not rich_text!)
		if entry.Properties.Progress.Number != nil {
			slugEntry.Progress = fmt.Sprintf("%.0f", *entry.Properties.Progress.Number)
		}
		// Handle image (files field)
		if len(entry.Properties.Image.Files) > 0 {
			slugEntry.Image, err = convertAndStoreImage(entry)
			if err != nil {
				log.Error("error converting and storing image: %v", err)
			}
		}
		// Handle comment (rich_text field that might be empty)
		if len(entry.Properties.Comment.RichText) > 0 {
			slugEntry.Comment = entry.Properties.Comment.RichText[0].PlainText
		}
		// append to slice
		readnowEntries = append(readnowEntries, slugEntry)
	}
	return readnowEntries, nil
}

func convertAndStoreImage(entry Entry) (string, error) {
	imageFile := entry.Properties.Image.Files[0]
	sourceURL := imageFile.External.URL
	if sourceURL == "" {
		sourceURL = imageFile.File.URL
	}
	if sourceURL == "" {
		return "", fmt.Errorf("no image URL found for entry %s", entry.ID)
	}

	resp, err := http.Get(sourceURL)
	if err != nil {
		return "", fmt.Errorf("error downloading image: %v", err)
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading image bytes: %v", err)
	}
	// store locally in ./images
	// Ensure the folder exists
	absPath, err := filepath.Abs("./images")
	if err != nil {
		return "", fmt.Errorf("error getting absolute path: %v", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		os.Mkdir(absPath, os.ModePerm)
	}

	var imageUrl string
	imageID := entry.ID
	filePath := filepath.Join(absPath, "/", imageID+".png")
	err = os.WriteFile(filePath, imageBytes, 0755)
	if err != nil {
		return "", fmt.Errorf("error writing image to file: %v", err)
	}

	imageUrl = "https://cloud.shaikzhafir.com/images/" + imageID + ".png"
	if os.Getenv("DEV") == "true" {
		imageUrl = "/images/" + imageID + ".png"
	}
	return imageUrl, nil
}

func (nc *notionClient) GetDatabaseID() string {
	return nc.DatabaseID
}

type converter struct {
	rawBlock []byte
	writer   io.Writer
	postType string
}

func extractLinkURL(link any) string {
	if linkMap, ok := link.(map[string]any); ok {
		if url, ok := linkMap["url"].(string); ok {
			return url
		}
	}
	return ""
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/bulleted_list_item.html")
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
	var contentHTML strings.Builder
	for _, richText := range block.BulletedListItem.Text {
		content := richText.PlainText
		if content == "" {
			content = richText.Text.Content
		}
		if content == "" {
			continue
		}

		segmentHTML := template.HTMLEscapeString(content)
		if richText.Annotations.Code {
			segmentHTML = `<code class="rounded bg-slate-200 px-1 py-0.5 font-mono text-[0.92em] text-red-600">` + segmentHTML + `</code>`
		}

		linkURL := richText.Href
		if linkURL == "" {
			linkURL = extractLinkURL(richText.Text.Link)
		}
		if linkURL != "" {
			escapedURL := template.HTMLEscapeString(linkURL)
			segmentHTML = `<a class="break-words text-sky-600 underline decoration-sky-300 underline-offset-4 transition-colors hover:text-sky-700" href="` + escapedURL + `" target="_blank" rel="noopener noreferrer">` + segmentHTML + `</a>`
		}
		contentHTML.WriteString(segmentHTML)
	}

	renderData := struct {
		Content template.HTML
	}{
		Content: template.HTML(contentHTML.String()),
	}

	err = tmpl.Execute(c.writer, renderData)
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/heading_1.html")
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/heading_2.html")
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/heading_3.html")
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/paragraph.html")
	if err != nil {
		return err
	}
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	var contentHTML strings.Builder
	for _, richText := range block.Paragraph.RichText {
		content := richText.PlainText
		if content == "" {
			content = richText.Text.Content
		}
		if content == "" {
			continue
		}

		segmentHTML := template.HTMLEscapeString(content)
		if richText.Annotations.Code {
			segmentHTML = `<code class="rounded bg-slate-200 px-1 py-0.5 font-mono text-[0.92em] text-red-600">` + segmentHTML + `</code>`
		}

		linkURL := richText.Href
		if linkURL == "" {
			linkURL = richText.Text.Link.URL
		}
		if linkURL != "" {
			escapedURL := template.HTMLEscapeString(linkURL)
			segmentHTML = `<a class="break-words text-sky-600 underline decoration-sky-300 underline-offset-4 transition-colors hover:text-sky-700" href="` + escapedURL + `" target="_blank" rel="noopener noreferrer">` + segmentHTML + `</a>`
		}
		contentHTML.WriteString(segmentHTML)
	}
	renderData := struct {
		Content template.HTML
	}{
		Content: template.HTML(contentHTML.String()),
	}
	err = tmpl.Execute(c.writer, renderData)
	if err != nil {
		return err
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/image.html")
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
	renderData := struct {
		Content  string
		PostType string
	}{
		Content:  block.Image.File.URL,
		PostType: c.postType,
	}
	log.Info("rendering image block with post type: %s", c.postType)
	err = tmpl.Execute(c.writer, renderData)
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
	templatePath, err := filepath.Abs("./templates/notion/blocks/code.html")
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

func NewConverter(writer io.Writer, rawBlock []byte, postType string) Converter {
	return &converter{
		rawBlock: rawBlock,
		writer:   writer,
		postType: postType,
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

func (nc *notionClient) ParseAndWriteNotionBlock(writer io.Writer, rawBlock []byte, postType string) error {
	// unmarshal to find block type
	var b models.Block
	err := json.Unmarshal(rawBlock, &b)
	if err != nil {
		return err
	}

	c := NewConverter(writer, rawBlock, postType)

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
