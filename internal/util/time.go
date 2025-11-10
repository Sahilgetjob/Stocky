package util

import "time"

var loc = time.FixedZone("Asia/Kolkata", 5*3600+1800)

func SetLocation(l *time.Location) {
	loc = l
}

func Now() time.Time {
	return time.Now().In(loc)
}

func TodayRange() (start, end time.Time) {
	n := Now()
	y, m, d := n.Date()
	start = time.Date(y, m, d, 0, 0, 0, 0, loc)
	end = start.Add(24 * time.Hour)
	return
}
