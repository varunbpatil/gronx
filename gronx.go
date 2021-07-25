package gronx

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var literals = strings.NewReplacer(
	"SUN", "0", "MON", "1", "TUE", "2", "WED", "3", "THU", "4", "FRI", "5", "SAT", "6",
	"JAN", "1", "FEB", "2", "MAR", "3", "APR", "4", "MAY", "5", "JUN", "6", "JUL", "7",
	"AUG", "8", "SEP", "9", "OCT", "10", "NOV", "11", "DEC", "12",
)

var expressions = map[string]string{
	"@yearly":    "0 0 1 1 *",
	"@annually":  "0 0 1 1 *",
	"@monthly":   "0 0 1 * *",
	"@weekly":    "0 0 * * 0",
	"@daily":     "0 0 * * *",
	"@hourly":    "0 * * * *",
	"@always":    "* * * * *",
	"@5minutes":  "*/5 * * * *",
	"@10minutes": "*/10 * * * *",
	"@15minutes": "*/15 * * * *",
	"@30minutes": "0,30 * * * *",
}

const (
	PosMinute = iota
	PosHour
	PosDayOfMonth
	PosMonth
	PosDayOfWeek
	PosYear
)

// SpaceRe is regex for whitespace.
var SpaceRe = regexp.MustCompile(`\s+`)

func normalize(expr string) []string {
	expr = strings.Trim(expr, " \t")
	if e, ok := expressions[strings.ToLower(expr)]; ok {
		expr = e
	}

	expr = SpaceRe.ReplaceAllString(expr, " ")
	expr = literals.Replace(strings.ToUpper(expr))

	return strings.Split(strings.ReplaceAll(expr, "  ", " "), " ")
}

// Gronx is the main program.
type Gronx struct {
	C Checker
}

// New initializes Gronx with factory defaults.
func New() Gronx {
	return Gronx{&SegmentChecker{}}
}

// IsDue checks if cron expression is due for given reference time (or now).
// It returns bool or error if any.
func (g *Gronx) IsDue(expr string, ref ...time.Time) (bool, error) {
	if len(ref) > 0 {
		g.C.SetRef(ref[0])
	} else {
		g.C.SetRef(time.Now())
	}

	segs, err := Segments(expr)
	if err != nil {
		return false, err
	}

	return g.SegmentsDue(segs)
}

// Segments splits expr into array array of cron parts.
// It returns array or error.
func Segments(expr string) ([]string, error) {
	segs := normalize(expr)
	if len(segs) < 5 || len(segs) > 6 {
		return []string{}, errors.New("expr should contain 5-6 segments separated by space")
	}

	return segs, nil
}

// SegmentsDue checks if all cron parts are due.
// It returns bool. You should use IsDue(expr) instead.
func (g *Gronx) SegmentsDue(segments []string) (bool, error) {
	for pos, seg := range segments {
		if seg == "*" || seg == "?" {
			continue
		}

		if due, err := g.C.CheckDue(seg, pos); !due {
			return due, err
		}
	}

	return true, nil
}

// IsValid checks if cron expression is valid.
// It returns bool.
func (g *Gronx) IsValid(expr string) bool {
	_, err := g.IsDue(expr)

	return err == nil
}

// GetPrev returns the previous time that the cron expression was due.
func (g *Gronx) GetPrev(expr string, ref ...time.Time) (*time.Time, error) {
	if len(ref) > 0 {
		g.C.SetRef(ref[0])
	} else {
		g.C.SetRef(time.Now())
	}

	segs, err := Segments(expr)
	if err != nil {
		return nil, err
	}

	// Maximum number of years we will check to find the previous due date.
	yearsLeftToCheck := 100

L:
	// See https://stackoverflow.com/a/322058 for the algorithm to calculate the previous due date.
	for yearsLeftToCheck > 0 {
		for _, pos := range []int{PosYear, PosMonth, PosDayOfMonth, PosDayOfWeek, PosHour, PosMinute} {
			if pos >= len(segs) {
				continue
			}
			seg := segs[pos]
			if seg != "*" && seg != "?" {
				due, err := g.C.CheckDue(seg, pos)
				if err != nil {
					return nil, err
				}
				if !due {
					prev, yearChanged := getPrevTime(g.C.GetRef(), pos)
					g.C.SetRef(prev)
					if yearChanged {
						yearsLeftToCheck -= 1
					}
					continue L
				}
			}
		}
		break
	}

	if yearsLeftToCheck == 0 {
		return nil, fmt.Errorf("could not find previous due for cron expression")
	}

	// Remove the second and nanosecond portion of the time before returning.
	res := g.C.GetRef()
	prevDue := time.Date(res.Year(), res.Month(), res.Day(), res.Hour(), res.Minute(), 0, 0, res.Location())

	return &prevDue, nil
}

func getPrevTime(ref time.Time, pos int) (time.Time, bool) {
	var res time.Time

	switch pos {
	case PosYear:
		res = time.Date(ref.Year(), 1, 1, 0, 0, 0, 0, ref.Location()).Add(-time.Nanosecond)
	case PosMonth:
		res = time.Date(ref.Year(), ref.Month(), 1, 0, 0, 0, 0, ref.Location()).Add(-time.Nanosecond)
	case PosDayOfMonth:
		res = time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(-time.Nanosecond)
	case PosDayOfWeek:
		res = time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, ref.Location()).Add(-time.Nanosecond)
	case PosHour:
		res = time.Date(ref.Year(), ref.Month(), ref.Day(), ref.Hour(), 0, 0, 0, ref.Location()).Add(-time.Nanosecond)
	case PosMinute:
		res = time.Date(ref.Year(), ref.Month(), ref.Day(), ref.Hour(), ref.Minute(), 0, 0, ref.Location()).Add(-time.Nanosecond)
	default:
		panic("Unknown segment position")
	}

	return res, res.Year() != ref.Year()
}
