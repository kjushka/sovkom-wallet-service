package currency_helpers

import (
	"fmt"
	"strings"
	"time"
)

const (
	CustomTimeLayout = "2006-01-02"
)

var nilTime = (time.Time{}).UnixNano()

type CustomTime struct {
	time.Time
}

func (ct *CustomTime) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	if s == "null" {
		ct.Time = time.Time{}
		return
	}
	ct.Time, err = time.Parse(CustomTimeLayout, s)
	if err == nil {
		return
	}

	return
}

func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Time.UnixNano() == nilTime {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf("\"%s\"", ct.Time.Format(CustomTimeLayout))), nil
}

func (ct *CustomTime) IsSet() bool {
	return ct.UnixNano() != nilTime
}
