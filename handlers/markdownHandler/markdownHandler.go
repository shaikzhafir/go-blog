package markdownHandler

import (
	"bytes"
	"html/template"
	log "htmx-blog/logging"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

type BlogPost struct {
	Title        string
	Slug         string
	PublishedStr string
	Published    time.Time
	Content      template.HTML
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

		blogPosts := []BlogPost{}
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
				log.Error(err.Error(), err)
			}
			document := markdown.Parser().Parse(text.NewReader(mdfile))
			metaData := document.OwnerDocument().Meta()
			title := metaData["Title"].(string)
			slug := metaData["Slug"].(string)
			pubDate := metaData["Published"].(string)
			pubDateTime, err := time.Parse("2-1-2006", pubDate)
			if err != nil {
				log.Error(err.Error(), err)
				pubDateTime = time.Now()
			}

			blogPosts = append(blogPosts, BlogPost{
				Title:     title,
				Slug:      slug,
				Published: pubDateTime,
				Content:   template.HTML(mdfile),
			})

			return nil
		})

		// sort the list of reviews by date
		// write the list of reviews to the page
		// return the page

		tmpl, err := template.ParseFiles("./templates/blogEntries.html", "./templates/blogEntry.html")
		if err != nil {
			w.Write([]byte(err.Error()))
		}

		sort.Slice(blogPosts, func(i, j int) bool {
			return blogPosts[i].Published.After(blogPosts[j].Published)
		})

		for i := range blogPosts {
			blogPosts[i].PublishedStr = blogPosts[i].Published.Format("2-January-2006")
		}

		err = tmpl.Execute(w, blogPosts)
		if err != nil {
			w.Write([]byte(err.Error()))
		}
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
		markdown := goldmark.New(
			goldmark.WithRendererOptions(
				html.WithUnsafe(),
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
				log.Error(err.Error(), err)
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
				log.Error(err.Error(), err)
			}
			log.Info("title: %v", title)

			review := ReviewData{
				Title:     title.(string),
				Slug:      slug.(string),
				Published: metaData["Published"].(string),
				Content:   template.HTML(buf.String()),
			}

			tmpl, err := template.ParseFiles("./templates/blogPost.html")
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
