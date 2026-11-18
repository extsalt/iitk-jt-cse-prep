package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	mrand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var mu sync.Mutex

type Change struct {
	T float64 `json:"t"`
	C string  `json:"c"`
}

type Resp struct {
	Choice  string   `json:"choice"`
	Marked  bool     `json:"marked"`
	Visits  int      `json:"visits"`
	Seconds float64  `json:"seconds"`
	History []Change `json:"history"`
}

type Event struct {
	T float64 `json:"t"`
	K string  `json:"k"`
}

type Attempt struct {
	ID           string           `json:"id"`
	Bank         string           `json:"bank"`
	Mode         string           `json:"mode"`
	Minutes      int              `json:"minutes"`
	Started      time.Time        `json:"started"`
	Submitted    *time.Time       `json:"submitted,omitempty"`
	Answers      map[string]*Resp `json:"answers"`
	Events       []Event          `json:"events"`
	PartCSeconds float64          `json:"partc_seconds"`
	Result       *Result          `json:"result,omitempty"`
}

type Tally struct {
	Correct int     `json:"correct"`
	Wrong   int     `json:"wrong"`
	Blank   int     `json:"blank"`
	Score   float64 `json:"score"`
	Seconds float64 `json:"seconds"`
}

func (t *Tally) add(status string, sec float64) {
	switch status {
	case "correct":
		t.Correct++
		t.Score++
	case "wrong":
		t.Wrong++
		t.Score -= 0.25
	default:
		t.Blank++
	}
	t.Seconds += sec
}

type TopicRow struct {
	Area  string `json:"area"`
	Topic string `json:"topic"`
	Tally
}

type QRes struct {
	ID          string  `json:"id"`
	Part        string  `json:"part"`
	Topic       string  `json:"topic"`
	Choice      string  `json:"choice"`
	First       string  `json:"first"`
	Answer      string  `json:"answer"`
	Status      string  `json:"status"`
	Seconds     float64 `json:"seconds"`
	Changes     int     `json:"changes"`
	Marked      bool    `json:"marked"`
	Explanation string  `json:"explanation"`
	Source      string  `json:"source,omitempty"`
}

type Result struct {
	Total          Tally             `json:"total"`
	Parts          map[string]*Tally `json:"parts"`
	Topics         []TopicRow        `json:"topics"`
	Questions      []QRes            `json:"questions"`
	NegLost        float64           `json:"neg_lost"`
	RightToWrong   int               `json:"right_to_wrong"`
	WrongToRight   int               `json:"wrong_to_right"`
	WrongToWrong   int               `json:"wrong_to_wrong"`
	ElapsedSeconds float64           `json:"elapsed_seconds"`
	Late           bool              `json:"late"`
	Blurs          int               `json:"blurs"`
}

func status(choice, answer string) string {
	switch {
	case choice == "":
		return "blank"
	case choice == answer:
		return "correct"
	}
	return "wrong"
}

func (a *App) score(at *Attempt, now time.Time) *Result {
	b := a.banks[at.Bank]
	r := &Result{Parts: map[string]*Tally{"A": {}, "B": {}}}
	topics := map[string]*TopicRow{}
	for i := range b.Questions {
		q := &b.Questions[i]
		resp := at.Answers[q.ID]
		if resp == nil {
			resp = &Resp{}
		}
		st := status(resp.Choice, q.Answer)
		qr := QRes{ID: q.ID, Part: q.Part, Topic: q.Topic, Choice: resp.Choice, Answer: q.Answer, Status: st,
			Seconds: math.Round(resp.Seconds*10) / 10, Marked: resp.Marked, Explanation: q.Explanation, Source: q.Source}
		var picks []string
		for _, h := range resp.History {
			if h.C != "" {
				picks = append(picks, h.C)
			}
		}
		if len(picks) > 0 {
			qr.First = picks[0]
		}
		qr.Changes = max(len(picks)-1, 0)
		if qr.First != "" && resp.Choice != "" && qr.First != resp.Choice {
			switch {
			case qr.First == q.Answer:
				r.RightToWrong++
			case resp.Choice == q.Answer:
				r.WrongToRight++
			default:
				r.WrongToWrong++
			}
		}
		r.Questions = append(r.Questions, qr)
		r.Total.add(st, resp.Seconds)
		r.Parts[q.Part].add(st, resp.Seconds)
		tr := topics[q.Topic]
		if tr == nil {
			tr = &TopicRow{Area: areaOf(q.Topic), Topic: q.Topic}
			topics[q.Topic] = tr
		}
		tr.add(st, resp.Seconds)
	}
	r.NegLost = float64(r.Total.Wrong) * 0.25
	for _, t := range topics {
		r.Topics = append(r.Topics, *t)
	}
	sortTopics(r.Topics)
	r.ElapsedSeconds = math.Round(now.Sub(at.Started).Seconds())
	r.Late = r.ElapsedSeconds > float64(at.Minutes*60+60)
	for _, e := range at.Events {
		if e.K == "blur" {
			r.Blurs++
		}
	}
	return r
}

func areaRank(area string) int {
	for i, a := range areas {
		if a.Name == area {
			return i
		}
	}
	return len(areas)
}

func sortTopics(t []TopicRow) {
	sort.Slice(t, func(i, j int) bool {
		if ai, aj := areaRank(t[i].Area), areaRank(t[j].Area); ai != aj {
			return ai < aj
		}
		return t[i].Topic < t[j].Topic
	})
}

func newID(prefix string) string {
	b := make([]byte, 3)
	rand.Read(b)
	return prefix + "-" + time.Now().Format("20060102-1504") + "-" + hex.EncodeToString(b)
}

func (a *App) attemptPath(id string) string {
	return filepath.Join(a.dir, "data", "attempts", filepath.Base(id)+".json")
}

func (a *App) loadAttempt(id string) (*Attempt, error) {
	raw, err := os.ReadFile(a.attemptPath(id))
	if err != nil {
		return nil, err
	}
	var at Attempt
	err = json.Unmarshal(raw, &at)
	return &at, err
}

func (a *App) saveAttempt(at *Attempt) error {
	b, _ := json.MarshalIndent(at, "", " ")
	tmp := a.attemptPath(at.ID) + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, a.attemptPath(at.ID))
}

func (a *App) allAttempts() []*Attempt {
	files, _ := filepath.Glob(filepath.Join(a.dir, "data", "attempts", "*.json"))
	var out []*Attempt
	for _, f := range files {
		if at, err := a.loadAttempt(strings.TrimSuffix(filepath.Base(f), ".json")); err == nil {
			out = append(out, at)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Started.Before(out[j].Started) })
	return out
}

type PracticeEntry struct {
	T       time.Time `json:"t"`
	Session string    `json:"session"`
	QID     string    `json:"qid"`
	Choice  string    `json:"choice"`
	Correct bool      `json:"correct"`
	Seconds float64   `json:"seconds"`
}

func (a *App) practicePath() string { return filepath.Join(a.dir, "data", "practice-log.jsonl") }

func (a *App) practiceLog() []PracticeEntry {
	f, err := os.Open(a.practicePath())
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []PracticeEntry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var e PracticeEntry
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			out = append(out, e)
		}
	}
	return out
}

// Outcome is one answer to one question, from an exam or from practice.
type Outcome struct {
	T       time.Time
	QID     string
	Choice  string
	Correct bool
	Seconds float64
	Source  string
}

func (a *App) outcomes() []Outcome {
	var out []Outcome
	for _, at := range a.allAttempts() {
		if at.Result == nil {
			continue
		}
		for _, q := range at.Result.Questions {
			if q.Choice != "" {
				out = append(out, Outcome{*at.Submitted, q.ID, q.Choice, q.Status == "correct", q.Seconds, "exam " + at.ID})
			}
		}
	}
	for _, e := range a.practiceLog() {
		if a.byID[e.QID] != nil {
			out = append(out, Outcome{e.T, e.QID, e.Choice, e.Correct, e.Seconds, "practice"})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].T.Before(out[j].T) })
	return out
}

type StatRow struct {
	Area     string  `json:"area"`
	Topic    string  `json:"topic"`
	Total    int     `json:"total"`
	Seen     int     `json:"seen"`
	Answers  int     `json:"answers"`
	Correct  int     `json:"correct"`
	Wrong    int     `json:"wrong"`
	Accuracy float64 `json:"accuracy"`
	AvgSec   float64 `json:"avg_seconds"`
	OpenMist int     `json:"open_mistakes"`
}

type Mistake struct {
	Q          PublicQ `json:"q"`
	Answer     string  `json:"answer"`
	Explain    string  `json:"explanation"`
	LastChoice string  `json:"last_choice"`
	TimesWrong int     `json:"times_wrong"`
	TimesRight int     `json:"times_right"`
	Last       string  `json:"last"`
}

type Stats struct {
	Topics   []StatRow `json:"topics"`
	Mistakes []Mistake `json:"mistakes"`
}

func (a *App) stats() Stats {
	type qs struct {
		wrong, right int
		last         Outcome
	}
	per := map[string]*qs{}
	rows := map[string]*StatRow{}
	row := func(topic string) *StatRow {
		if rows[topic] == nil {
			rows[topic] = &StatRow{Area: areaOf(topic), Topic: topic}
		}
		return rows[topic]
	}
	for _, id := range a.order {
		for _, q := range a.banks[id].Questions {
			row(q.Topic).Total++
		}
	}
	for _, o := range a.outcomes() {
		q := a.byID[o.QID]
		s := per[o.QID]
		if s == nil {
			s = &qs{}
			per[o.QID] = s
		}
		s.last = o
		r := row(q.Topic)
		r.Answers++
		r.AvgSec += o.Seconds
		if o.Correct {
			s.right++
			r.Correct++
		} else {
			s.wrong++
			r.Wrong++
		}
	}
	st := Stats{Topics: []StatRow{}, Mistakes: []Mistake{}}
	for qid, s := range per {
		q := a.byID[qid]
		r := row(q.Topic)
		r.Seen++
		if !s.last.Correct {
			r.OpenMist++
			st.Mistakes = append(st.Mistakes, Mistake{a.public(q), q.Answer, q.Explanation, s.last.Choice, s.wrong, s.right, s.last.T.Format("2 Jan 15:04")})
		}
	}
	for _, r := range rows {
		if r.Answers > 0 {
			r.Accuracy = math.Round(float64(r.Correct)*1000/float64(r.Answers)) / 10
			r.AvgSec = math.Round(r.AvgSec/float64(r.Answers)*10) / 10
		}
		st.Topics = append(st.Topics, *r)
	}
	sort.Slice(st.Topics, func(i, j int) bool {
		if ai, aj := areaRank(st.Topics[i].Area), areaRank(st.Topics[j].Area); ai != aj {
			return ai < aj
		}
		return st.Topics[i].Topic < st.Topics[j].Topic
	})
	sort.Slice(st.Mistakes, func(i, j int) bool {
		if st.Mistakes[i].TimesWrong != st.Mistakes[j].TimesWrong {
			return st.Mistakes[i].TimesWrong > st.Mistakes[j].TimesWrong
		}
		return st.Mistakes[i].Q.ID < st.Mistakes[j].Q.ID
	})
	return st
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	http.Error(w, msg, code)
}

func (a *App) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/banks", func(w http.ResponseWriter, r *http.Request) {
		type row struct {
			ID, Title string
			Minutes   int
			Count     int
			PartA     int
			PartB     int
		}
		out := []row{}
		for _, id := range a.order {
			b := a.banks[id]
			rw := row{ID: id, Title: b.Title, Minutes: b.Minutes, Count: len(b.Questions)}
			for _, q := range b.Questions {
				if q.Part == "A" {
					rw.PartA++
				} else {
					rw.PartB++
				}
			}
			out = append(out, rw)
		}
		type area struct {
			Name   string   `json:"name"`
			Topics []string `json:"topics"`
		}
		var ar []area
		have := map[string]bool{}
		for _, q := range a.byID {
			have[q.Topic] = true
		}
		for _, x := range areas {
			var ts []string
			for _, t := range x.Topics {
				if have[t] {
					ts = append(ts, t)
				}
			}
			if len(ts) > 0 {
				ar = append(ar, area{x.Name, ts})
			}
		}
		writeJSON(w, map[string]any{"banks": out, "areas": ar})
	})

	mux.HandleFunc("POST /api/exams", func(w http.ResponseWriter, r *http.Request) {
		var req struct{ Bank, Mode string }
		json.NewDecoder(r.Body).Decode(&req)
		b := a.banks[req.Bank]
		if b == nil {
			fail(w, 404, "no such bank")
			return
		}
		at := &Attempt{ID: newID(b.ID), Bank: b.ID, Mode: "full", Minutes: b.Minutes, Started: time.Now(), Answers: map[string]*Resp{}}
		if req.Mode == "ab" {
			at.Mode, at.Minutes = "ab", b.Minutes-35
		}
		mu.Lock()
		err := a.saveAttempt(at)
		mu.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]string{"id": at.ID})
	})

	mux.HandleFunc("GET /api/exams/{id}", func(w http.ResponseWriter, r *http.Request) {
		at, err := a.loadAttempt(r.PathValue("id"))
		if err != nil {
			fail(w, 404, "no such attempt")
			return
		}
		b := a.banks[at.Bank]
		var qs []PublicQ
		for i := range b.Questions {
			qs = append(qs, a.public(&b.Questions[i]))
		}
		out := map[string]any{"attempt": at, "title": b.Title, "questions": qs, "now": time.Now()}
		if at.Mode == "full" {
			pcs := make([]PartC, len(b.PartC))
			for i, p := range b.PartC {
				pcs[i] = PartC{ID: p.ID, Title: p.Title, Words: p.Words, Prompt: p.Prompt}
				if at.Submitted != nil {
					pcs[i].Guide = p.Guide
				}
			}
			out["partc"] = pcs
			if at.Submitted != nil {
				out["partc_note"] = b.PartCNote
			}
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("PUT /api/exams/{id}", func(w http.ResponseWriter, r *http.Request) {
		a.updateExam(w, r, false)
	})
	mux.HandleFunc("POST /api/exams/{id}/submit", func(w http.ResponseWriter, r *http.Request) {
		a.updateExam(w, r, true)
	})

	mux.HandleFunc("GET /api/exams", func(w http.ResponseWriter, r *http.Request) {
		type row struct {
			ID, Bank, Title, Mode string
			Started               time.Time
			Submitted             *time.Time
			Score                 *float64
			A, B                  *Tally
		}
		out := []row{}
		for _, at := range a.allAttempts() {
			rw := row{ID: at.ID, Bank: at.Bank, Mode: at.Mode, Started: at.Started, Submitted: at.Submitted}
			if b := a.banks[at.Bank]; b != nil {
				rw.Title = b.Title
			}
			if at.Result != nil {
				rw.Score, rw.A, rw.B = &at.Result.Total.Score, at.Result.Parts["A"], at.Result.Parts["B"]
			}
			out = append([]row{rw}, out...)
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("GET /api/practice", func(w http.ResponseWriter, r *http.Request) {
		want := map[string]bool{}
		for _, t := range strings.Split(r.URL.Query().Get("topics"), ",") {
			if t != "" {
				want[t] = true
			}
		}
		mode := r.URL.Query().Get("mode")
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n <= 0 {
			n = 20
		}
		seen := map[string]bool{}
		for _, o := range a.outcomes() {
			seen[o.QID] = true
		}
		open := map[string]bool{}
		if mode == "mistakes" {
			for _, m := range a.stats().Mistakes {
				open[m.Q.ID] = true
			}
		}
		pool := []PublicQ{}
		for _, id := range a.order {
			for i := range a.banks[id].Questions {
				q := &a.banks[id].Questions[i]
				if len(want) > 0 && !want[q.Topic] {
					continue
				}
				if mode == "new" && seen[q.ID] || mode == "mistakes" && !open[q.ID] {
					continue
				}
				pool = append(pool, a.public(q))
			}
		}
		mrand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		if len(pool) > n {
			pool = pool[:n]
		}
		writeJSON(w, map[string]any{"session": newID("practice"), "questions": pool})
	})

	mux.HandleFunc("POST /api/practice/answer", func(w http.ResponseWriter, r *http.Request) {
		var e PracticeEntry
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			fail(w, 400, err.Error())
			return
		}
		q := a.byID[e.QID]
		if q == nil || len(e.Choice) != 1 {
			fail(w, 400, "bad question or choice")
			return
		}
		e.T, e.Correct = time.Now(), e.Choice == q.Answer
		line, _ := json.Marshal(e)
		mu.Lock()
		f, err := os.OpenFile(a.practicePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			_, err = f.Write(append(line, '\n'))
			f.Close()
		}
		mu.Unlock()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"correct": e.Correct, "answer": q.Answer, "explanation": q.Explanation, "source": q.Source})
	})

	mux.HandleFunc("GET /api/stats", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, a.stats())
	})
}

func (a *App) updateExam(w http.ResponseWriter, r *http.Request, submit bool) {
	var in struct {
		Answers      map[string]*Resp `json:"answers"`
		Events       []Event          `json:"events"`
		PartCSeconds float64          `json:"partc_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, 400, err.Error())
		return
	}
	mu.Lock()
	defer mu.Unlock()
	at, err := a.loadAttempt(r.PathValue("id"))
	if err != nil {
		fail(w, 404, "no such attempt")
		return
	}
	if at.Submitted != nil {
		fail(w, 409, "already submitted")
		return
	}
	b := a.banks[at.Bank]
	valid := map[string]bool{}
	for _, q := range b.Questions {
		valid[q.ID] = true
	}
	at.Answers = map[string]*Resp{}
	for id, resp := range in.Answers {
		if valid[id] && resp != nil && (resp.Choice == "" || len(resp.Choice) == 1 && resp.Choice[0] >= 'a' && resp.Choice[0] <= 'd') {
			at.Answers[id] = resp
		}
	}
	at.Events, at.PartCSeconds = in.Events, in.PartCSeconds
	if submit {
		now := time.Now()
		at.Submitted = &now
		at.Result = a.score(at, now)
	}
	if err := a.saveAttempt(at); err != nil {
		fail(w, 500, err.Error())
		return
	}
	writeJSON(w, map[string]any{"ok": true, "submitted": submit})
}

func fmtSec(s float64) string {
	return fmt.Sprintf("%d:%02d", int(s)/60, int(s)%60)
}
