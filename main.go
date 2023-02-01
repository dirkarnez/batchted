package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/antchfx/jsonquery"
	"github.com/asticode/go-astisub"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var (
	input string
	start int
	end   int
)

type Entry struct {
	URL        string `json:"url"`
	Transcript string `json:"transcript"`
	Summary    string `json:"summary"`
}

func main() {
	flag.IntVar(&start, "start", -1, "start index (inclusive)")
	flag.IntVar(&end, "end", -1, "end index (exclusive)")
	flag.StringVar(&input, "input", "", "input file to work on")
	flag.Parse()

	if len(input) < 1 {
		// create context
		opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
		ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
		defer cancel()

		ctx, cancel = chromedp.NewContext(
			ctx,
			chromedp.WithLogf(log.Printf))
		defer cancel()

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

		input = fmt.Sprintf("%s.txt", strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-"))

		err = ioutil.WriteFile(input, bytes, 0644)
		checkErr(err)

		fmt.Println("Done,", count)
	}

	total := []Entry{}
	bytes, err := ioutil.ReadFile(input)
	checkErr(err)

	json.Unmarshal(bytes, &total)

	var _start_inclusive int = 0
	var _end_exclusive int = len(total)

	if start > -1 {
		_start_inclusive = start
	}

	if end > -1 {
		_end_exclusive = end
	}

	newTotal := []Entry{}
	client := &http.Client{}
	for i, item := range total[_start_inclusive:_end_exclusive] {
		subtitleURL, err := getSubtitleURL(item.URL)
		checkErr(err)
		if len(subtitleURL) > 0 {
			content, err := DownloadVTT(subtitleURL, client)
			checkErr(err)
			fmt.Println(i, item.URL, subtitleURL)

			newTotal = append(newTotal, Entry{URL: item.URL, Transcript: content})
		}
	}

	newBytes, err := json.MarshalIndent(newTotal, "", "\t")
	checkErr(err)

	err = ioutil.WriteFile(input, newBytes, 0644)
	checkErr(err)
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

func getSubtitleURL(urlstr string) (string, error) {
	// create context
	done := make(chan bool)

	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var subtitleURL string
	chromedp.ListenTarget(ctx, func(ev interface{}) {

		switch ev := ev.(type) {

		case *network.EventResponseReceived:
			resp := ev.Response
			if strings.Contains(resp.URL, "metadata.json") && resp.MimeType == "application/json" {
				doc, _ := jsonquery.LoadURL(resp.URL)
				if doc != nil {
					subtitle := jsonquery.FindOne(doc, "/subtitles/*[code='en']/webvtt")
					if subtitle != nil && len(subtitleURL) < 1 {
						subtitleURL = fmt.Sprintf("%s", subtitle.Value())
					}
				}
			}
		}
	})

	// pageCtx is used to open the page
	err := chromedp.Run(ctx,
		chromedp.Navigate(urlstr),
		chromedp.WaitVisible(`#fdgfdgfdgfdg`, chromedp.ByID),
	)
	if err != nil {
		close(done)
	}

	<-done

	// if err == context.DeadlineExceeded {
	//     return subtitleURL, nil
	// }

	return subtitleURL, nil
}

func DownloadVTT(webvttURL string, client *http.Client) (string, error) {
	resp, err := client.Get(webvttURL)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	s2, err := astisub.ReadFromWebVTT(bytes.NewReader(body))
	if err != nil {
		return "", err
	}

	// file, err := os.Create("example.txt")
	// if err != nil {
	// 	return err
	// }

	// defer file.Close()
	buf := bytes.NewBufferString("")

	//w := bufio.NewWriter(file)

	for _, item := range s2.Items {
		fmt.Fprintf(buf, "%s ", item.String())
	}
	return buf.String(), nil
}
