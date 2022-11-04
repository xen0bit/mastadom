package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-mastodon"
	"github.com/xen0bit/mastadom/pkg/dbtools"
	"golang.org/x/net/html"
)

var (
	mc *mastodon.Client
)

func htmlContentToText(htmlContent string) string {
	fmt.Println(htmlContent)

	tkn := html.NewTokenizer(strings.NewReader(htmlContent))
	var textContent string

	for {
		tt := tkn.Next()
		switch {
		case tt == html.ErrorToken:
			return textContent
		case tt == html.EndTagToken:
			t := tkn.Token()
			if t.Data == "p" {
				textContent += "\n"
			}
		case tt == html.SelfClosingTagToken:
			t := tkn.Token()
			if t.Data == "br" {
				textContent += "\n"
			}
		case tt == html.TextToken:
			t := tkn.Token()
			textContent += t.Data
		}
	}
}

func filterTimeline(timeline []*mastodon.Status) (filteredTimeline []mastodon.Status) {
	// Only want to add certain types of text to pipeline
	for i := len(timeline) - 1; i >= 0; i-- {
		toAdd := true
		item := timeline[i]
		if item.Sensitive {
			toAdd = false
		}
		if len(item.MediaAttachments) != 0 {
			toAdd = false
		}
		if len(item.SpoilerText) != 0 {
			toAdd = false
		}
		if len(item.Mentions) != 0 {
			toAdd = false
		}

		// If pass filter, append to output
		if toAdd {
			filteredTimeline = append(filteredTimeline, *item)
		}
	}
	return filteredTimeline
}

func timelineScraper(ch chan dbtools.SqliteRow) {
	timeline, err := mc.GetTimelinePublic(context.Background(), true, nil)
	if err != nil {
		log.Fatal(err)
	}
	filteredTimeline := filterTimeline(timeline)
	for i := len(filteredTimeline) - 1; i >= 0; i-- {
		ch <- dbtools.SqliteRow{
			Id:      string(timeline[i].ID),
			Content: htmlContentToText(timeline[i].Content),
		}
	}
	close(ch)
}

func main() {
	// Sqlite Setup
	sqliteconn := dbtools.NewSqliteConn("mastadom.db")
	defer sqliteconn.DB.Close()
	if err := sqliteconn.CreateTables(); err != nil {
		log.Fatal(err)
	}

	// Load config from env
	mServer := os.Getenv("MASTADOM_SERVER")
	mClientId := os.Getenv("MASTADOM_CLIENTID")
	mClientSecret := os.Getenv("MASTADOM_CLIENTSECRET")
	mEmail := os.Getenv("MASTADOM_EMAIL")
	mPassword := os.Getenv("MASTADOM_PASSWORD")

	// Mastadon global client
	mc = mastodon.NewClient(&mastodon.Config{
		Server:       mServer,
		ClientID:     mClientId,
		ClientSecret: mClientSecret,
	})
	err := mc.Authenticate(context.Background(), mEmail, mPassword)
	if err != nil {
		log.Fatal(err)
	}

	// Loop with sleep
	for {
		ch := make(chan dbtools.SqliteRow)
		// Producer
		go timelineScraper(ch)
		// Consumer
		if err := sqliteconn.InsertData(ch); err != nil {
			log.Fatal(err)
		}
		time.Sleep(60 * time.Second)
	}
}
