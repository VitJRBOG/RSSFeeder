package parser

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

//
// NOTE
// Parser for https://www.nationalgeographic.com/pages/topic/latest-stories
//

type Article struct {
	Link        string
	Date        string
	Title       string
	Description string
}

func (a *Article) composeInfo(articleTag *html.Node) {
	a.extractTitle(articleTag)
	a.Link = articleTag.Attr[3].Val

	doc, err := fetchHTMLNode(a.Link)
	if err != nil {
		log.Printf("\n%s\n%s", err.Error(), debug.Stack())
		return
	}

	a.extractDescription(doc)
	a.extractPublicationDate(doc)
}

func (a *Article) extractTitle(tag *html.Node) {
	title := tag.Attr[2].Val
	i := strings.LastIndex(title, ",")

	a.Title = title[:i]
}

func (a *Article) extractDescription(doc *html.Node) {
	tag := findTag("Article__Headline__Desc", doc)

	if tag == nil {
		return
	}

	var err error
	a.Description, err = getTextFromTag(tag)
	if err != nil {
		log.Print(err.Error())
	}

	i := strings.Index(a.Description, ">")
	a.Description = a.Description[i+1:]

	i = strings.Index(a.Description, "<")
	a.Description = a.Description[:i]
}

func (a *Article) extractPublicationDate(doc *html.Node) {
	tag := findTag("Byline__Meta Byline__Meta--publishDate", doc)

	var err error
	a.Date, err = getTextFromTag(tag)
	if err != nil {
		log.Print(err.Error())
	}

	if len([]rune(a.Date)) <= 0 {
		return
	}

	i := strings.Index(a.Date, ">")
	a.Date = a.Date[i+1:]

	i = strings.Index(a.Date, "<")
	a.Date = a.Date[:i]

	a.Date = strings.ReplaceAll(a.Date, "Published ", "")
}

func GetArticles(u string) ([]*Article, error) {
	doc, err := fetchHTMLNode(u)
	if err != nil {
		return []*Article{}, err
	}

	externalTag := findTag("FilterBar", doc).NextSibling

	articles := composeArticles(externalTag)

	return articles, nil
}

func findTag(tagName string, doc *html.Node) *html.Node {
	return getElementByClass(doc, tagName)
}

func composeArticles(externalTag *html.Node) []*Article {
	var articles []*Article
	var wg sync.WaitGroup

	articles = extractTagAttributes(&wg, articles,
		findTag("GridPromoTile__Row", externalTag.FirstChild))
	articles = extractTagAttributes(&wg, articles,
		findTag("GridPromoTile__Row", externalTag.FirstChild.NextSibling.NextSibling))

	wg.Wait()

	return articles
}

func extractTagAttributes(wg *sync.WaitGroup, articles []*Article,
	externalTag *html.Node) []*Article {
	commonTag := externalTag.FirstChild
	for {
		if commonTag == nil {
			break
		}

		if findTag("SectionLabel--sponsor", commonTag) == nil {
			asdas := findTag("SectionLabel SectionLabel--link", commonTag)
			text, err := getTextFromTag(asdas.FirstChild.FirstChild)
			if err != nil {
				log.Print(err.Error())
			}
			if !(strings.Contains(text, "Magazine")) {
				articleTag := commonTag.FirstChild.FirstChild.FirstChild
				var a Article
				articles = append(articles, &a)
				wg.Add(1)
				go func(a *Article) {
					a.composeInfo(articleTag)
					wg.Done()
				}(&a)
			}
		}

		commonTag = commonTag.NextSibling
	}

	return articles
}

func getTextFromTag(doc *html.Node) (string, error) {
	var buf bytes.Buffer
	w := io.Writer(&buf)
	err := html.Render(w, doc)
	if err != nil {
		return "", fmt.Errorf("\n%s\n%s", err.Error(), debug.Stack())
	}

	return buf.String(), nil
}

func getElementByClass(n *html.Node, k string) *html.Node {
	return traverse(n, k)
}

func traverse(n *html.Node, k string) *html.Node {
	if checkKey(n, k) {
		return n
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result := traverse(c, k)
		if result != nil {
			return result
		}
	}

	return nil
}

func checkKey(n *html.Node, k string) bool {
	if n.Type == html.ElementNode {
		s, ok := getAttribute(n, "class")
		if ok && strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func getAttribute(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

func fetchHTMLNode(u string) (*html.Node, error) {
	body, err := sendRequest(u)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("\n%s\n%s", err.Error(), debug.Stack())
	}

	return doc, nil
}

func sendRequest(u string) ([]byte, error) {
	response, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("\n%s\n%s", err.Error(), debug.Stack())
	}

	defer func() {
		if err != nil {
			log.Printf("\n%s\n%s", err, debug.Stack())
		}
	}()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("\n%s\n%s", err.Error(), debug.Stack())
	}

	return body, nil
}
