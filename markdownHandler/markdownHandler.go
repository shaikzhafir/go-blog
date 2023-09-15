package markdownHandler

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

type MarkdownHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetReviewsList() http.HandlerFunc
	GetReviewByTitle() http.HandlerFunc
	GetBlogList() http.HandlerFunc
}

func NewHandler() MarkdownHandler {
	return &markdownHandler{}
}

type markdownHandler struct {
}

func (h *markdownHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// extract the path from the request
	path := r.URL.Path

	// fetch html from service layer
	// for now just return some random html page with the path
	html := "<html><body><h1> test" + path + "</h1></body></html>"
	w.Write([]byte(html))
}

func (h *markdownHandler) GetBlogList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		html := "<html><body><h1> test" + "blogposts" + "</h1></body></html>"
		w.Write([]byte(html))
	}
}

func (h *markdownHandler) GetReviewsList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		markdown := goldmark.New(
			goldmark.WithExtensions(
				meta.New(
					meta.WithStoresInDocument(),
				),
			),
		)
		filepath.WalkDir("./reviews", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				w.Write([]byte(err.Error()))
				return err
			}
			if d.IsDir() {
				return nil
			}
			mdfile, err := os.ReadFile(path)
			if err != nil {
				fmt.Println(err.Error())
			}
			document := markdown.Parser().Parse(text.NewReader(mdfile))
			metaData := document.OwnerDocument().Meta()
			title := metaData["Title"]
			slug := metaData["Slug"]
			pubDate := metaData["Published"]
			w.Write([]byte(fmt.Sprintf(`
			<div class="my-2">
			<a href="/reviews/%s">
			<p class="font-bold text-2xl">%v</p>
			<p>%v</p>
			</a>
			</div>
			`, slug, title, pubDate)))
			return nil
		})
	}
}

type ReviewData struct {
	Title     string
	Slug      string
	Published string
	Content   template.HTML
}

func (h *markdownHandler) GetReviewByTitle() http.HandlerFunc {
	// make a map of title to path
	// iterate through map and find the path
	// read the file at that path
	// convert the file to html
	// return the html
	return func(w http.ResponseWriter, r *http.Request) {
		fullPath := r.URL.Path
		segments := strings.Split(fullPath, "/")
		currentSlug := segments[len(segments)-1]
		fmt.Printf("slug: %v\n", currentSlug)
		markdown := goldmark.New(
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
			goldmark.WithExtensions(
				meta.New(
					meta.WithStoresInDocument(),
				),
			),
		)
		filepath.WalkDir("./reviews", func(path string, d os.DirEntry, err error) error {
			if err != nil {
				w.Write([]byte(err.Error()))
				return err
			}
			if d.IsDir() {
				return nil
			}
			mdfile, err := os.ReadFile(path)
			if err != nil {
				fmt.Println(err.Error())
			}

			document := markdown.Parser().Parse(text.NewReader(mdfile))
			metaData := document.OwnerDocument().Meta()

			title := metaData["Title"]
			slug := metaData["Slug"]
			if slug != currentSlug {
				return nil
			}

			var buf bytes.Buffer
			err = markdown.Convert(mdfile, &buf)
			if err != nil {
				fmt.Println(err.Error())
			}
			fmt.Println(buf.String())

			review := ReviewData{
				Title:     title.(string),
				Slug:      slug.(string),
				Published: metaData["Published"].(string),
				Content:   template.HTML(buf.String()),
			}

			/* htmlToDisplay := fmt.Sprintf(`
			<div>
			<a href="/">take me back this post hurts my eyes</a>
			<h2>%s</h2>
			<h3 class="mb-10">%s</h3>
			`, title, metaData["Published"])

			htmlToDisplay += buf.String()
			htmlToDisplay += "</div>"

			w.Write([]byte(htmlToDisplay)) */

			tmpl, err := template.ParseFiles("./templates/blog.html")
			if err != nil {
				w.Write([]byte(err.Error()))
			}

			err = tmpl.Execute(w, review)
			if err != nil {
				w.Write([]byte(err.Error()))
			}

			return nil
		})
	}
}

func (h *markdownHandler) TestRoute(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}

func mdToHTML(source []byte) []byte {
	// create markdown parser with extensions
	gmd := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithBlockParsers(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)
	var buf bytes.Buffer
	if err := gmd.Convert(source, &buf); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func splitMetadataAndContent(markdownText string) (metadata string, content string) {
	lines := strings.Split(markdownText, "\n")

	// Check for metadata delimiter "---"
	if len(lines) >= 3 && lines[0] == "---" {
		// Extract metadata
		metadata = strings.Join(lines[1:2], "\n")
		// Extract content
		content = strings.Join(lines[3:], "\n")
	} else {
		// No metadata found, consider the entire document as content
		content = markdownText
	}

	return metadata, content
}
