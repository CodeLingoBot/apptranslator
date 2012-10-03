package main

// TODO: add an option to log to a file in the format:
// $time E: $msg
// $time N: $msg
// E: is for errors, N: is for notices
// format of $time is TBD (human readable is long, unix timestamp is short
// but not human-readable)

// TODO: gather all errors and email them periodically (e.g. every day) to myself

import (
	"fmt"
	"time"
)

type TimestampedMsg struct {
	Time time.Time
	Msg  string
}

type CircularMessagesBuf struct {
	Msgs []TimestampedMsg
	pos  int
	full bool
}

func NewCircularMessagesBuf(cap int) *CircularMessagesBuf {
	return &CircularMessagesBuf{
		Msgs: make([]TimestampedMsg, cap, cap),
		pos:  0,
		full: false,
	}
}

func (b *CircularMessagesBuf) Add(s string) {
	var msg = TimestampedMsg{time.Now(), s}
	if b.pos == cap(b.Msgs) {
		b.pos = 0
		b.full = true
	}
	b.Msgs[b.pos] = msg
	b.pos += 1
}

/*
func reverseInPlace(arr []*TimestampedMsg) {
	j := len(arr) - 1
	for i, tmp := range arr {
		if i >= j {
			break
		}
		arr[i] = arr[j]
		arr[j] = tmp
		j -= 1
	}
}
*/

func (b *CircularMessagesBuf) GetOrdered() []*TimestampedMsg {
	size := b.pos
	if b.full {
		size = cap(b.Msgs)
	}
	res := make([]*TimestampedMsg, size, size)
	for i := 0; i < size; i++ {
		p := b.pos - 1 - i
		if p < 0 {
			p = cap(b.Msgs) + p
		}
		res[i] = &b.Msgs[p]
	}
	return res
}

type ServerLogger struct {
	Errors  *CircularMessagesBuf
	Notices *CircularMessagesBuf
}

func NewServerLogger(errorsMax, noticesMax int) *ServerLogger {
	l := &ServerLogger{
		Errors:  NewCircularMessagesBuf(errorsMax),
		Notices: NewCircularMessagesBuf(noticesMax),
	}
	return l
}

func (l *ServerLogger) Error(s string) {
	l.Errors.Add(s)
}

func (l *ServerLogger) Errorf(format string, v ...interface{}) {
	l.Errors.Add(fmt.Sprintf(format, v...))
}

func (l *ServerLogger) Notice(s string) {
	l.Notices.Add(s)
}

func (l *ServerLogger) Noticef(format string, v ...interface{}) {
	l.Notices.Add(fmt.Sprintf(format, v...))
}

func (l *ServerLogger) GetErrors() []*TimestampedMsg {
	return l.Errors.GetOrdered()
}

func (l *ServerLogger) GetNotices() []*TimestampedMsg {
	return l.Notices.GetOrdered()
}
