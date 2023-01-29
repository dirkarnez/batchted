// package main

// import (
// 	"bufio"
// 	"bytes"
// 	"fmt"
// 	"io/ioutil"
// 	"log"
// 	"net/http"
// 	"os"

// 	astisub "github.com/asticode/go-astisub"
// )

// func main() {
// 	client := &http.Client{}

// 	resp, err := client.Get("https://hls.ted.com/project_masters/8044/subtitles/en/full.vtt?intro_master_id=2346")
// 	checkErr(err)

// 	defer resp.Body.Close()
// 	body, err := ioutil.ReadAll(resp.Body)
// 	checkErr(err)

// 	s2, err := astisub.ReadFromWebVTT(bytes.NewReader(body))
// 	checkErr(err)

// 	file, err := os.Create("example.txt")
// 	checkErr(err)

// 	defer file.Close()
// 	w := bufio.NewWriter(file)

// 	for _, item := range s2.Items {
// 		fmt.Fprintf(w, "%s ", item.String())
// 	}
// 	w.Flush()
// }

// func checkErr(err error) {
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/dslipak/pdf"
)

var (
	url string
)

func main() {
	flag.StringVar(&url, "url", "", "target url")
	flag.Parse()

	if len(url) < 1 {
		log.Fatalln("--url is needed")
		return
	}

	// create context
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf))
	defer cancel()

	// capture pdf
	var buf []byte
	if err := chromedp.Run(ctx, printToPDF(url, &buf)); err != nil {
		log.Fatal(err)
		return
	}

	if err := ioutil.WriteFile("sample.pdf", buf, 0o644); err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println("wrote sample.pdf")
}

// var body = document.body,
//     html = document.documentElement;

// var height = Math.max( body.scrollHeight, body.offsetHeight,
//                        html.clientHeight, html.scrollHeight, html.offsetHeight );

// print a specific pdf page.
func printToPDF(urlstr string, res *[]byte) chromedp.Tasks {
	tasks := chromedp.Tasks{}

	tasks = append(tasks,
		chromedp.Navigate(urlstr),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(0, 0)", nil),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER)", nil),
		// chromedp.Sleep(1*time.Second),
		// chromedp.WaitVisible(`document.querySelector("#js_content > section:nth-child(53) > section:nth-child(6) > img")`, chromedp.ByJSPath),
	)
	// var nodes []*cdp.Node
	tasks = append(tasks,
		// chromedp.Nodes(`document.querySelector("#js_content > section:nth-child(53) > section:nth-child(5) > p:nth-child(5)")`, &nodes, chromedp.ByJSPath),
		// chromedp.MouseClickNode(nodes[0], chromedp.ButtonType(input.Right)),
		chromedp.Sleep(5*time.Second),
		chromedp.EvaluateAsDevTools(`Array.from(document.body.getElementsByTagName("*"))
		.filter(element => element.style.overflow != "visible")
		.forEach(element => {
			element.style.overflow = "visible";
		})`, nil),
		chromedp.EvaluateAsDevTools(`Array.from(document.body.getElementsByTagName("pre"))
		.forEach(element => {
			element.style.whiteSpace = "pre-wrap";
			element.style.wordBreak = "break-word";
		})`, nil),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(0, 0)", nil),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER)", nil),
		// chromedp.WaitVisible(`document.querySelector("#js_content > section:nth-child(53) > section:nth-child(5) > p:nth-child(5) > img")`, chromedp.ByJSPath),
		// chromedp.Sleep(5*time.Second),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(0, 0)", nil),
		// chromedp.EvaluateAsDevTools("document.documentElement.scrollTo(Number.MAX_SAFE_INTEGER, Number.MAX_SAFE_INTEGER)", nil),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var height = 100.0
			var isMultiplePage = true

			for isMultiplePage {
				buf, _, err := page.PrintToPDF().WithPaperHeight(height).WithPrintBackground(true).Do(ctx)
				if err != nil {
					return err
				}

				reader := bytes.NewReader(buf)

				pdfReader, err := pdf.NewReader(reader, int64(len(buf)))
				if err != nil {
					return err
				}

				numPage := pdfReader.NumPage()
				if numPage == 1 {
					isMultiplePage = false
					*res = buf
				} else {
					height = height + 10
				}
			}

			return nil
		}),
	)

	return tasks
}
