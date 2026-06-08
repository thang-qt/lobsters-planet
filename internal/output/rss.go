package output

import (
	"encoding/xml"
	"strings"
	"time"

	"lobsters-planet/internal/config"
)

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	AtomNS  string     `xml:"xmlns:atom,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	AtomLink    atomLink  `xml:"atom:link"`
	LastBuild   string    `xml:"lastBuildDate"`
	Items       []rssItem `xml:"item"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	Description string  `xml:"description,omitempty"`
	Author      string  `xml:"author,omitempty"`
	PubDate     string  `xml:"pubDate,omitempty"`
}

type rssGUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

func writeRSS(path string, cfg config.Config, entries []PublicEntry, generatedAt time.Time) error {
	items := make([]rssItem, 0, len(entries))
	for _, entry := range entries {
		item := rssItem{
			Title:       entry.Title,
			Link:        entry.URL,
			GUID:        rssGUID{IsPermaLink: "false", Value: entry.ID},
			Description: entry.Summary,
			Author:      entry.OwnerUsername,
		}
		if t := entryTime(entry); !t.IsZero() {
			item.PubDate = t.Format(time.RFC1123Z)
		}
		items = append(items, item)
	}

	doc := rssDocument{
		Version: "2.0",
		AtomNS:  "http://www.w3.org/2005/Atom",
		Channel: rssChannel{
			Title:       cfg.Output.SiteTitle,
			Link:        cfg.Output.SiteURL,
			Description: "Latest posts from Lobste.rs users' personal sites.",
			AtomLink:    atomLink{Href: joinURL(cfg.Output.SiteURL, "feed.xml"), Rel: "self", Type: "application/rss+xml"},
			LastBuild:   generatedAt.Format(time.RFC1123Z),
			Items:       items,
		},
	}
	data, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append([]byte(xml.Header), data...)
	data = append(data, '\n')
	return writeFile(path, data)
}

func joinURL(base, path string) string {
	if base == "" {
		return path
	}
	return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(path, "/")
}
