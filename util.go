package main

import (
	"github.com/prometheus/common/log"
)

func check(e error) bool {
	if e != nil {
		log.Errorln(e)
		return true
	}
	return false
}
func empty(o []interface{}) bool {
	return o == nil || !(len(o) > 0)
}
