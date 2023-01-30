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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

var (
	url string
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// create context
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf))
	defer cancel()

	type Entry struct {
		URL        string `json:"url"`
		Transcript string `json:"transcript"`
		Summary    string `json:"summary"`
	}

	total := []Entry{}

	for i := 1; i < 145; i++ {
		var url = fmt.Sprintf("https://www.ted.com/talks?language=en&page=%d&sort=newest", i)
		// see Entry
		var outputJsonArrayString string
		if err := chromedp.Run(ctx, doStuff(url, &outputJsonArrayString)); err != nil {
			log.Fatal(err)
			return
		}
		var arr []Entry
		_ = json.Unmarshal([]byte(outputJsonArrayString), &arr)

		fmt.Println("page", i)
		total = append(total, arr...)
	}

	count := len(total)
	bytes, err := json.MarshalIndent(total, "", "\t")
	checkErr(err)

	filename := fmt.Sprintf("%s.txt", strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-"))

	err = ioutil.WriteFile(filename, bytes, 0644)
	checkErr(err)

	fmt.Println("Done,", count)
}

func doStuff(urlstr string, outputJsonArrayString *string) chromedp.Tasks {
	tasks := chromedp.Tasks{}
	tasks = append(tasks,
		chromedp.Navigate(urlstr),
		chromedp.WaitVisible(`document.querySelector("#browse-results")`, chromedp.ByJSPath),
		chromedp.EvaluateAsDevTools(`JSON.stringify(Array.from(document.getElementById("browse-results").getElementsByClassName("media__message")).map(msg => ({url: msg.getElementsByClassName("ga-link")[0].href})))`, &outputJsonArrayString),
	)

	return tasks
}

func getTitle(urlstr string) (string, error) {
	// create context
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf))
	defer cancel()

	var title string

	chromedp.ListenTarget(ctx, func(ev interface{}) {

		switch ev := ev.(type) {

		case *network.EventResponseReceived:
			resp := ev.Response
			log.Printf("received headers: %s %s", resp.URL, resp.MimeType)

			// if resp.URL == urlstr {
			// 	log.Printf("received headers: %s %s", resp.URL, resp.MimeType)
			// 	if resp.MimeType != "text/html" {
			// 		chromedp.Cancel(ctx)
			// 	}

			// 	if strings.Contains(resp.URL, "youtube.com") {
			// 		log.Printf("YT!!")
			// 	}

			// 	// may be redirected
			// 	switch ContentType := resp.Headers["Content-Type"].(type) {
			// 	case string:
			// 		// here v has type T
			// 		if !strings.Contains(ContentType, "text/html") {
			// 			chromedp.Cancel(ctx)
			// 		}
			// 	}

			// 	switch ContentType := resp.Headers["content-type"].(type) {
			// 	case string:
			// 		// here v has type T
			// 		if !strings.Contains(ContentType, "text/html") {
			// 			chromedp.Cancel(ctx)
			// 		}
			// 	}
			// }
		}
	})

	req := `
(async () => new Promise((resolve, reject) => {
	var handle = NaN;
	(function animate() {
		if (!isNaN(handle)) {
			clearTimeout(handle);
		}
		if (document.title.length > 0 && !document.title.startsWith("http")) {
			resolve(document.title);
		} else {
			handle = setTimeout(animate, 1000);
		}
	}());
}));
`
	err := chromedp.Run(ctx,
		chromedp.Navigate(urlstr),
		//chromedp.Evaluate(`window.location.href`, &res),
		chromedp.Evaluate(req, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams {
			return p.WithAwaitPromise(true)
		}),
		chromedp.Title(&title),
	)
	if err == context.Canceled {
		// url as title
		log.Printf("Cancel!!")
		return urlstr, nil
	}

	return title, err
}
