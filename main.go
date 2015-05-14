package main

import (
	//"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"log"
	"os"
	"strings"
)

func parse() {
	s := `<html><p>Links:</p><ul><li><a href="foo">Foo</a><li><a href="/bar/baz">BarBaz</a></ul></html>`
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		log.Fatal(err)
	}
	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			tag := &html.Node{}
			tag.DataAtom = atom.Script
			tag.Type = html.ElementNode
			tag.Data = "script"

			//Content
			c := &html.Node{
				Type: html.TextNode,
				Data: `alert("injected")`,
			}
			n.AppendChild(tag)
			tag.AppendChild(c)

			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	html.Render(os.Stdout, doc)
}

func main() {
	parse()
}
