package markdownHandler

import (
	"bytes"
	"io/ioutil"
	"net/http"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

type MarkdownHandler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	GetReviewsList() http.HandlerFunc
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
		mdfile, err := ioutil.ReadFile("./reviews/haha.md")
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		html := mdToHTML(mdfile)
		w.Write(html)
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
