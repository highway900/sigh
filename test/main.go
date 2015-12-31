package main

import (
    "fmt"
    "golang.org/x/net/html"
    "log"
    "strings"
    "bytes"
    "io/ioutil"
)

const WSSCRIPT string = `
    var serversocket = new WebSocket("ws://localhost:3000/ws/echo");
 
    serversocket.onopen = function() {
        serversocket.send("Connection init");
    }

    // Reload the page
    serversocket.onmessage = function(e) {
        if (e.data == "refresh") {
            document.location.reload(true);
        }
    };
`

func main() {
    buf := new(bytes.Buffer)
    
    content, err := ioutil.ReadFile("index.html")
    if err != nil {
        panic("Error: " + err.Error())
    }
    
    doc, err := html.Parse(strings.NewReader(string(content)))
    if err != nil {
        log.Fatal(err)
    }
    var f func(*html.Node)
    f = func(n *html.Node) {
        if n.Type == html.ElementNode && n.Data == "body" {
            // <script> tag
            tag := &html.Node{}
            tag.Type = html.ElementNode
            tag.Data = "script"

            // script Content/text
            c := &html.Node{
                Type: html.TextNode,
                Data: WSSCRIPT,
            }
            tag.AppendChild(c)
            n.AppendChild(tag)

            return
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            f(c)
        }
    }
    f(doc)

    html.Render(buf, doc)
    fmt.Println(buf.String())
}
