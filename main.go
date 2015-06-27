package main

import (
	"flag"
	"fmt"
	"github.com/go-fsnotify/fsnotify"
	"github.com/gorilla/websocket"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type handler struct {
	filename string
	content  string
	script   string
	c        chan bool
}

func (h *handler) watcher(f string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					h.c <- true
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(f)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func (h *handler) inject() {
	doc, err := html.Parse(strings.NewReader(h.content))
	if err != nil {
		log.Fatal(err)
	}
	var f func(n *html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			// <script> tag
			tag := &html.Node{}
			tag.Type = html.ElementNode
			tag.Data = "script"

			// script Content/text
			c := &html.Node{
				Type: html.TextNode,
				Data: h.script,
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
	html.Render(h, doc)
}

func (h *handler) Write(p []byte) (n int, err error) {
	h.content = string(p)
	fmt.Println(string(p))
	return len(h.content), nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Read in index.html
	// Inject WS script for refresh
	content, err := ioutil.ReadFile(h.filename)
	if err != nil {
		panic("Error: " + err.Error())
	}
	h.content = string(content)

	h.inject()
	fmt.Fprintf(w, h.content)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *handler) reloadHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Error:", err)
		return
	}

	for {
		refresh := <-h.c

		if refresh {
			err = conn.WriteMessage(1, []byte("refresh"))
			if err != nil {
				return
			}
		}
	}
}

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
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("filename not specified")
	}
	filename := flag.Args()[0]

	h := &handler{
		filename: filename,
		script:   WSSCRIPT,
		c:        make(chan bool),
	}

	// Watch file(s) for changes and trigger refresh
	go h.watcher(h.filename)

	http.HandleFunc("/ws/echo", h.reloadHandler)
	http.Handle("/", h)
	go http.ListenAndServe(":3000", nil)
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		panic("Error: " + err.Error())
	}

	fmt.Println("Done")
}
