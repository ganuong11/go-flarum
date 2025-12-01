package util

import (
	"strconv"
	"time"
)

const (
	TIME_FMT = "2006-01-02 15:04"
)

// TimeNow current time
func TimeNow() uint64 {
	return uint64(time.Now().UTC().Unix())
}

// TimeFmt format timestamp
func TimeFmt(tp interface{}, sample string, tz int) string {
	offset := int64(time.Duration(tz) * time.Hour)
	var t int64
	switch tp.(type) {
	case uint64:
		t = int64(tp.(uint64))
	case string:
		i64, err := strconv.ParseInt(tp.(string), 10, 64)
		if err != nil {
			return ""
		}
		t = i64
	case int64:
		t = tp.(int64)
	}
	if len(sample) == 0 {
		sample = "2006-01-02 15:04:05"
	}
	tm := time.Unix(t, offset).UTC()
	return tm.Format(sample)
}

// TimeHuman time for humans to see
func TimeHuman(ts interface{}) string {
	var t int64
	switch ts.(type) {
	case uint64:
		t = int64(ts.(uint64))
	case string:
		i64, err := strconv.ParseInt(ts.(string), 10, 64)
		if err != nil {
			return ""
		}
		t = i64
	case int64:
		t = ts.(int64)
	}

	then := time.Unix(t, 0)
	diff := time.Now().UTC().Sub(then)

	hours := diff.Hours()
	days := int(hours) / 24
	if days > 0 {
		switch {
		case days >= 365:
			y := days / 365
			d := days % 365
			if d == 0 {
				return strconv.Itoa(y) + " years ago"
			}
			return strconv.Itoa(y) + " year " + strconv.Itoa(d) + " days ago"
		case days >= 30:
			m := days / 30
			d := days % 30
			if d == 0 {
				return strconv.Itoa(m) + " months ago"
			}
			return strconv.Itoa(m) + " month " + strconv.Itoa(d) + " days ago"
		case days >= 7:
			w := days / 7
			d := days % 7
			if d == 0 {
				return strconv.Itoa(w) + " weeks ago"
			}
			return strconv.Itoa(w) + " week " + strconv.Itoa(d) + " days ago"
		case days >= 1:
			h := int(hours) % 24
			if h == 0 {
				return strconv.Itoa(days) + " days ago"
			}
			return strconv.Itoa(days) + " day " + strconv.Itoa(h) + " hours ago"
		default:
			return "1 day ago"
		}
	}

	seconds := diff.Seconds()
	switch {
	case seconds >= 3600:
		h := int(seconds / 3600)
		m := int(seconds) % 3600
		if m == 0 {
			return strconv.Itoa(h) + " hours ago"
		}
		return strconv.Itoa(h) + " hour " + strconv.Itoa(m) + " minutes ago"
	case seconds >= 60:
		m := int(seconds / 60)
		s := int(seconds) % 60
		if s == 0 {
			return strconv.Itoa(m) + " minutes ago"
		}
		return strconv.Itoa(m) + " min " + strconv.Itoa(s) + " seconds ago"
	}

	return "just now"
}
