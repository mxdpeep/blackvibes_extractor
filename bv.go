package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type Poem struct {
	ID       int      `json:"id"`
	File     string   `json:"file"`
	Title    string   `json:"title"`
	Date     string   `json:"date"`
	Lines    []string `json:"lines"`
	Hashtags string   `json:"hashtags"`
	Mood     string   `json:"mood"`
	Music    string   `json:"music"`
	Sfx      string   `json:"sfx"`
	Vfx      string   `json:"vfx"`
}

func cleanString(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func main() {
	root := "."
	var allPoems []Poem

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			p := extractPoem(path)
			if p != nil {
				p.File = filepath.Base(path)
				allPoems = append(allPoems, *p)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}

	for i := range allPoems {
		allPoems[i].ID = i + 1
		allPoems[i].Hashtags = ""
		allPoems[i].Mood = ""
		allPoems[i].Music = ""
		allPoems[i].Sfx = ""
		allPoems[i].Vfx = ""
	}
	exportToCSV(allPoems, "poems.csv")

	rand.Shuffle(len(allPoems), func(i, j int) {
		allPoems[i], allPoems[j] = allPoems[j], allPoems[i]
	})

	for i := range allPoems {
		allPoems[i].ID = i + 1
	}
	outputJSON, _ := json.MarshalIndent(allPoems, "", "  ")
	os.WriteFile("poems.json", outputJSON, 0644)
	fmt.Printf("Vyexportováno %d básní.\n", len(allPoems))
}

func exportToCSV(poems []Poem, filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Nelze vytvořit CSV: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write([]string{"id", "title", "date", "content"})
	for _, p := range poems {
		content := strings.Join(p.Lines, "\n")
		writer.Write([]string{
			strconv.Itoa(p.ID),
			p.Title,
			p.Date,
			content,
		})
	}
}

func extractPoem(path string) *Poem {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	doc, err := goquery.NewDocumentFromReader(file)
	if err != nil {
		return nil
	}

	var foundPoem *Poem

	doc.Find(".post-body").Each(func(i int, s *goquery.Selection) {
		title := doc.Find("h3.post-title, .post-title, .entry-title").First().Text()
		if title == "" {
			title = "Bez názvu"
		}

		dateHeader := doc.Find("h2.date-header span, .date-header span, .date-header").First().Text()

		timeText := ""
		doc.Find(".post-footer, .post-timestamp").Each(func(j int, footer *goquery.Selection) {
			txt := footer.Text()
			if strings.Contains(txt, "v ") {
				parts := strings.Split(txt, "v ")
				if len(parts) > 1 {
					timeFields := strings.Fields(strings.TrimSpace(parts[1]))
					if len(timeFields) >= 1 {
						timeText = "v " + timeFields[0]
					}
				}
			}
		})
		finalDate := cleanString(dateHeader + " " + timeText)

		s.Find("*").Each(func(i int, sel *goquery.Selection) {
			if sel.Is("br") {
				sel.ReplaceWithHtml(" @BR@ ")
			} else if sel.Is("div") || sel.Is("p") {
				if strings.TrimSpace(sel.Text()) != "" || sel.Find("br").Length() > 0 {
					sel.AppendHtml(" @BR@ ")
				}
			} else {
				sel.PrependHtml(" ")
				sel.AppendHtml(" ")
			}
		})

		rawText := s.Text()
		rawText = strings.ReplaceAll(rawText, "@BR@", "\n")
		lines := strings.Split(rawText, "\n")

		var cleanLines []string
		consecutiveEmpties := 0
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "Vystavil") || strings.Contains(trimmed, "Autor:") || strings.Contains(trimmed, "reakce:") {
				continue
			}
			if trimmed == "" {
				consecutiveEmpties++
			} else {
				trimmed = strings.ReplaceAll(trimmed, "-", "–")
				trimmed = strings.ReplaceAll(trimmed, "...", "…")
				if consecutiveEmpties >= 2 && len(cleanLines) > 0 {
					cleanLines = append(cleanLines, "")
				}
				cleanLines = append(cleanLines, trimmed)
				consecutiveEmpties = 0
			}
		}
		for len(cleanLines) > 0 && cleanLines[0] == "" {
			cleanLines = cleanLines[1:]
		}
		for len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] == "" {
			cleanLines = cleanLines[:len(cleanLines)-1]
		}
		if len(cleanLines) > 0 {
			foundPoem = &Poem{
				Title: cleanString(title),
				Date:  finalDate,
				Lines: cleanLines,
			}
		}
	})

	return foundPoem
}
