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
)

type Entry struct {
	URL        string `json:"url"`
	Transcript string `json:"transcript"`
	Summary    string `json:"summary"`
}

func main() {
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

	newTotal := []Entry{}
	client := &http.Client{}
	for i, item := range total {
		subtitleURL, err := getSubtitleURL(item.URL)
		checkErr(err)
		if len(subtitleURL) > 0 {
			content, err := DownloadVTT(subtitleURL, client)
			checkErr(err)
			fmt.Println(i, item.URL, subtitleURL) //, content)

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
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", false))
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf))
	defer cancel()

	var subtitleURL string

	chromedp.ListenTarget(ctx, func(ev interface{}) {

		switch ev := ev.(type) {

		case *network.EventResponseReceived:
			resp := ev.Response

			if strings.Contains(resp.URL, "metadata.json") && resp.MimeType == "application/json" {
				doc, err := jsonquery.LoadURL(resp.URL)
				checkErr(err)
				subtitle := jsonquery.FindOne(doc, "/subtitles/*[name='English']/webvtt")
				subtitleURL = fmt.Sprintf("%s", subtitle.Value())
			}
		}
	})

	err := chromedp.Run(ctx,
		chromedp.Navigate(urlstr),
	)

	return subtitleURL, err
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
