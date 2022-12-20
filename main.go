package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	astisub "github.com/asticode/go-astisub"
)

func main() {
	client := &http.Client{}

	resp, err := client.Get("https://hls.ted.com/project_masters/8044/subtitles/en/full.vtt?intro_master_id=2346")
	checkErr(err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	checkErr(err)

	s2, err := astisub.ReadFromWebVTT(bytes.NewReader(body))
	checkErr(err)

	file, err := os.Create("example.txt")
	checkErr(err)

	defer file.Close()
	w := bufio.NewWriter(file)

	for _, item := range s2.Items {
		fmt.Fprintf(w, "%s ", item.String())
	}
	w.Flush()
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
