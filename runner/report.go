package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func short(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// report prints, as plain text, every exam taken, the latest exam in detail, practice totals, topic statistics and open mistakes.
func (a *App) report(w io.Writer) {
	all := a.allAttempts()
	var done []*Attempt
	fmt.Fprintln(w, "== Exams")
	for _, at := range all {
		if at.Result == nil {
			fmt.Fprintf(w, "%s  %s  not submitted (started %s)\n", at.ID, at.Mode, at.Started.Format("2 Jan 15:04"))
			continue
		}
		done = append(done, at)
		r := at.Result
		fmt.Fprintf(w, "%s  %s  A %.2f (%d/%d/%d)  B %.2f (%d/%d/%d)  total %.2f/100  time %s%s\n", at.ID, at.Mode,
			r.Parts["A"].Score, r.Parts["A"].Correct, r.Parts["A"].Wrong, r.Parts["A"].Blank,
			r.Parts["B"].Score, r.Parts["B"].Correct, r.Parts["B"].Wrong, r.Parts["B"].Blank,
			r.Total.Score, fmtSec(r.ElapsedSeconds), map[bool]string{true: "  LATE", false: ""}[r.Late])
	}
	fmt.Fprintln(w, "   (correct/wrong/blank)")
	if len(done) > 0 {
		at := done[len(done)-1]
		r := at.Result
		fmt.Fprintf(w, "\n== Latest exam %s (%s, submitted %s)\n", at.ID, a.banks[at.Bank].Title, at.Submitted.Format("2 Jan 15:04"))
		fmt.Fprintf(w, "Negative marks lost: %.2f.  Answer changes: right->wrong %d, wrong->right %d, wrong->wrong %d.  Window left %d times.  Part C tab: %s\n",
			r.NegLost, r.RightToWrong, r.WrongToRight, r.WrongToWrong, r.Blurs, fmtSec(at.PartCSeconds))
		fmt.Fprintf(w, "Time on A: %s, on B: %s\n", fmtSec(r.Parts["A"].Seconds), fmtSec(r.Parts["B"].Seconds))
		fmt.Fprintln(w, "\nTopic                      C  W  B  score  time")
		for _, t := range r.Topics {
			if t.Wrong+t.Blank > 0 {
				fmt.Fprintf(w, "%-25s %2d %2d %2d %6.2f %5s\n", short(t.Topic, 25), t.Correct, t.Wrong, t.Blank, t.Score, fmtSec(t.Seconds))
			}
		}
		fmt.Fprintln(w, "(topics with no wrong or blank answer are left out)")
		fmt.Fprintln(w, "\nWrong and blank:")
		for _, q := range r.Questions {
			if q.Status == "correct" {
				continue
			}
			qq := a.byID[q.ID]
			ch := q.Choice
			if ch == "" {
				ch = "-"
			}
			fmt.Fprintf(w, "  %s [%s] %s  you %s, key %s, %.0fs, changes %d  | %s\n", q.ID, q.Topic, q.Status, ch, q.Answer, q.Seconds, q.Changes, short(qq.Question, 70))
		}
		qs := append([]QRes(nil), r.Questions...)
		sort.Slice(qs, func(i, j int) bool { return qs[i].Seconds > qs[j].Seconds })
		fmt.Fprintln(w, "\nSlowest 10:")
		for _, q := range qs[:min(10, len(qs))] {
			fmt.Fprintf(w, "  %s [%s] %.0fs %s\n", q.ID, q.Topic, q.Seconds, q.Status)
		}
	}
	pl := a.practiceLog()
	right := 0
	days := map[string]int{}
	for _, e := range pl {
		if e.Correct {
			right++
		}
		days[e.T.Format("2006-01-02")]++
	}
	fmt.Fprintf(w, "\n== Practice: %d answers, %d correct", len(pl), right)
	var dk []string
	for d := range days {
		dk = append(dk, d)
	}
	sort.Strings(dk)
	for _, d := range dk {
		fmt.Fprintf(w, "; %s: %d", d, days[d])
	}
	fmt.Fprintln(w)
	st := a.stats()
	fmt.Fprintln(w, "\n== Topics across exams and practice (answers, accuracy, avg time, open mistakes; seen/total)")
	for _, t := range st.Topics {
		if t.Answers == 0 {
			fmt.Fprintf(w, "%-25s not practised yet (%d questions)\n", short(t.Topic, 25), t.Total)
			continue
		}
		fmt.Fprintf(w, "%-25s %3d  %5.1f%%  %5.1fs  open %d  seen %d/%d\n", short(t.Topic, 25), t.Answers, t.Accuracy, t.AvgSec, t.OpenMist, t.Seen, t.Total)
	}
	fmt.Fprintf(w, "\n== Open mistakes (latest answer wrong): %d\n", len(st.Mistakes))
	for _, m := range st.Mistakes {
		fmt.Fprintf(w, "  %s [%s] wrong %dx, right %dx, last chose %s (key %s) %s | %s\n", m.Q.ID, m.Q.Topic, m.TimesWrong, m.TimesRight, m.LastChoice, m.Answer, m.Last, short(m.Q.Question, 60))
	}
	fmt.Fprintf(w, "\nReport made %s\n", time.Now().Format("2 Jan 2006 15:04"))
}
