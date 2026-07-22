// Command synth-ui serves the workbench with counters that survive a restart.
//
// It is the same page as `synth ui`; the only difference is where the totals in
// the header are kept. `synth ui` counts in memory and forgets on exit, because
// the core library has two dependencies and a SQLite driver is ten more.
//
//	synth-ui                       # ~/.config/synth/stats.db
//	synth-ui --port 9000
//	synth-ui --db ./project.db     # counters for one project
//	synth-ui --recent 20           # print the last runs and exit
//
// The database holds counts and never a generated value, and nothing is sent
// anywhere. The server binds loopback only, exactly as `synth ui` does.
package main

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/bakhodir/synth/ui"
	"github.com/bakhodir/synth/ui/stats"
)

func main() {
	port := flag.String("port", "8080", "port to serve on (loopback only)")
	path := flag.String("db", "", "counter database (default: ~/.config/synth/stats.db)")
	recent := flag.Int("recent", 0, "print the last N runs and exit")
	flag.Parse()

	db, err := stats.Open(*path)
	if err != nil {
		fail(err)
	}
	defer db.Close()

	if *recent > 0 {
		if err := printRecent(db, *recent); err != nil {
			fail(err)
		}
		return
	}

	totals, err := db.Totals()
	if err != nil {
		fail(err)
	}
	fmt.Printf("counters: %s (%d runs, %d rows so far)\n", db.Path(), totals.Files, totals.Rows)

	if err := ui.Serve("127.0.0.1:"+*port, ui.WithRecorder(db)); err != nil {
		fail(err)
	}
}

func printRecent(db *stats.DB, n int) error {
	runs, err := db.Recent(n)
	if err != nil {
		return err
	}
	if len(runs) == 0 {
		fmt.Println("no runs recorded yet")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "WHEN\tNAME\tROWS\tCOLS\tFORMAT\tBYTES\tMS")
	for _, r := range runs {
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%s\t%d\t%d\n",
			r.At.Format(time.RFC3339), r.Name, r.Rows, r.Columns, r.Format, r.Bytes, r.Millis)
	}
	return w.Flush()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "synth-ui:", err)
	os.Exit(1)
}
