// Command runner is a local CBT and practice app for the question banks in banks/*.json.
//
//	go run .            serve on http://127.0.0.1:7070
//	go run . report     print the latest exam and the topic statistics (for review)
//	go run . check      validate every bank
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed web
var webFS embed.FS

type Question struct {
	ID          string    `json:"id"`
	Part        string    `json:"part"`
	Topic       string    `json:"topic"`
	Passage     string    `json:"passage,omitempty"`
	Question    string    `json:"question"`
	Code        string    `json:"code,omitempty"`
	Options     [4]string `json:"options"`
	Answer      string    `json:"answer"`
	Explanation string    `json:"explanation"`
	Source      string    `json:"source,omitempty"`
}

type PartC struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Words  int    `json:"words"`
	Prompt string `json:"prompt"`
	Guide  string `json:"guide"`
}

type Bank struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Minutes   int        `json:"minutes"`
	Questions []Question `json:"questions"`
	PartC     []PartC    `json:"partc"`
	PartCNote string     `json:"partc_note"`
}

// PublicQ is a question without its answer, sent to the browser before it is answered.
type PublicQ struct {
	ID       string    `json:"id"`
	Bank     string    `json:"bank"`
	Part     string    `json:"part"`
	Topic    string    `json:"topic"`
	Area     string    `json:"area"`
	Passage  string    `json:"passage,omitempty"`
	Question string    `json:"question"`
	Code     string    `json:"code,omitempty"`
	Options  [4]string `json:"options"`
}

type App struct {
	dir    string
	banks  map[string]*Bank
	order  []string
	byID   map[string]*Question
	bankOf map[string]string
}

func (a *App) public(q *Question) PublicQ {
	return PublicQ{q.ID, a.bankOf[q.ID], q.Part, q.Topic, areaOf(q.Topic), q.Passage, q.Question, q.Code, q.Options}
}

func loadBanks(dir string) (*App, []string, error) {
	a := &App{dir: dir, banks: map[string]*Bank{}, byID: map[string]*Question{}, bankOf: map[string]string{}}
	files, _ := filepath.Glob(filepath.Join(dir, "banks", "*.json"))
	sort.Strings(files)
	var problems []string
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}
		var b Bank
		if err := json.Unmarshal(raw, &b); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", f, err)
		}
		if b.ID == "" {
			b.ID = strings.TrimSuffix(filepath.Base(f), ".json")
		}
		if b.Title == "" {
			b.Title = b.ID
		}
		for i := range b.Questions {
			q := &b.Questions[i]
			where := fmt.Sprintf("%s question %d (%s)", b.ID, i+1, q.ID)
			switch {
			case q.ID == "":
				problems = append(problems, where+": no id")
				continue
			case a.byID[q.ID] != nil:
				problems = append(problems, where+": duplicate id")
				continue
			}
			if len(q.Answer) != 1 || q.Answer[0] < 'a' || q.Answer[0] > 'd' {
				problems = append(problems, where+": answer must be a, b, c or d")
			}
			if q.Part != "A" && q.Part != "B" {
				problems = append(problems, where+": part must be A or B")
			}
			if areaOf(q.Topic) == "" {
				problems = append(problems, where+fmt.Sprintf(": topic %q is not listed in topics.go", q.Topic))
			}
			if strings.TrimSpace(q.Question) == "" {
				problems = append(problems, where+": empty question")
			}
			for k, o := range q.Options {
				if strings.TrimSpace(o) == "" {
					problems = append(problems, where+fmt.Sprintf(": option %c is empty", 'a'+k))
				}
			}
			if strings.TrimSpace(q.Explanation) == "" {
				problems = append(problems, where+": no explanation")
			}
			a.byID[q.ID] = q
			a.bankOf[q.ID] = b.ID
		}
		bb := b
		a.banks[b.ID] = &bb
		a.order = append(a.order, b.ID)
	}
	return a, problems, nil
}

func main() {
	addr := flag.String("addr", "127.0.0.1:7070", "listen address")
	dir := flag.String("dir", ".", "folder that holds banks/ and data/")
	flag.Parse()
	a, problems, err := loadBanks(*dir)
	if err != nil {
		log.Fatal(err)
	}
	cmd := flag.Arg(0)
	if cmd == "check" || len(problems) > 0 {
		for _, p := range problems {
			fmt.Println("problem:", p)
		}
		for _, id := range a.order {
			fmt.Printf("%s: %d questions, %d Part C\n", id, len(a.banks[id].Questions), len(a.banks[id].PartC))
		}
		if len(problems) > 0 {
			os.Exit(1)
		}
		if cmd == "check" {
			return
		}
	}
	if cmd == "report" {
		a.report(os.Stdout)
		return
	}
	os.MkdirAll(filepath.Join(*dir, "data", "attempts"), 0755)
	sub, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(sub)))
	a.routes(mux)
	fmt.Printf("Runner: open http://%s  (Ctrl+C to stop)\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
