package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/gocolly/colly"
	"gitlab.com/merfrei/colly-project/pkg/config"
	mongoM "gitlab.com/merfrei/colly-project/pkg/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Article extracted
type Article struct {
	Identifier    string   `json:"identifier"`
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	Author        string   `json:"author"`
	PublishedDate string   `json:"publishedDate"`
	Claim         string   `json:"claim"`
	ClaimDate     string   `json:"claimDate"`
	Rating        string   `json:"rating"`
	Tags          []string `json:"tags"`
	Sources       []string `json:"sources"`
}

const limitPages = 1

func main() {
	configFile := flag.String("config", "config.toml", "Path to the config file")
	collectionName := flag.String("collection", "politifact", "The collection when the items will be stored")

	flag.Parse()

	configD, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	config := config.Load(configD)

	mongoCli, err := mongoM.Connect(config.Mongo.URI)
	if err != nil {
		log.Fatal(err)
	}
	defer mongoM.Disconnect(mongoCli)

	db := mongoCli.Database(config.Mongo.Database)
	col := db.Collection(*collectionName)

	c := colly.NewCollector(
		colly.Async(true),
		colly.AllowedDomains("politifact.com", "www.politifact.com"),
	)

	c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 4})

	articleCollector := c.Clone()

	// Before making a request print "Visiting ..."
	c.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	// On every a element which has href attribute call callback
	c.OnHTML(".m-statement__quote a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		articleURL := e.Request.AbsoluteURL(link)
		log.Println("Link found:", articleURL)
		articleCollector.Visit(articleURL)
	})

	itemChan := make(chan Article)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		opts := options.FindOneAndUpdate().SetUpsert(true)
		for article := range itemChan {
			filter := bson.D{{Key: "identifier", Value: article.Identifier}}
			update := bson.D{{Key: "$set", Value: article}}
			updatedDocument := &Article{}
			err = col.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(updatedDocument)
			if err != nil {
				if err == mongo.ErrNoDocuments {
					log.Printf("New Article => %+v\n", article)
				} else {
					log.Println(err)
				}
				continue
			}
			log.Printf("Updated Article => %+v\n", updatedDocument)
		}
	}()

	sdre := regexp.MustCompile(`\s(\w+\s\d+,\s\d+)\s`)

	articleCollector.OnHTML("main", func(h *colly.HTMLElement) {
		url := h.Request.URL.String()
		log.Println("Article found", url)
		splits := strings.Split(url, "/")
		var identifier string
		for i := len(splits) - 1; i >= 0; i-- {
			if splits[i] != "" {
				identifier = splits[i]
				break
			}
		}
		article := Article{
			Identifier:    identifier,
			URL:           url,
			Title:         h.ChildText("h2.c-title"),
			Author:        h.ChildText(".m-author__content > a"),
			PublishedDate: h.ChildText(".m-author__date"),
			Claim:         strings.TrimSpace(h.DOM.Find(".m-statement__quote-wrap > .m-statement__quote").First().Text()),
			Rating:        h.ChildAttr(".m-statement__meter .c-image__original", "alt"),
			Tags:          []string{},
			Sources:       []string{},
		}
		dateStr := h.ChildText(".m-statement__desc")
		matches := sdre.FindStringSubmatch(dateStr)
		if len(matches) > 1 {
			article.ClaimDate = matches[1]
		}
		h.ForEach("ul.m-list.m-list--horizontal a.c-tag", func(_ int, a *colly.HTMLElement) {
			article.Tags = append(article.Tags, a.ChildText("span"))
		})
		h.ForEach("#sources article p", func(_ int, s *colly.HTMLElement) {
			article.Sources = append(article.Sources, s.Text)
		})

		itemChan <- article
	})

	c.Visit("https://www.politifact.com/factchecks/")

	c.Wait()
	articleCollector.Wait()

	close(itemChan)

	wg.Wait()
}
