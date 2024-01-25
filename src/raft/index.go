package raft

import (
	"log"
	"math"
)

func (rf *Raft) getSlice(startIndex, endIndex int) []LogEntry {
	logEntries := rf.log

	logStartIndex := startIndex
	logEndIndex := endIndex
	if logStartIndex <= 0 {
		Debug(dError, "LogEntries.getSlice: startIndex out of range. startIndex: %d, len: %d.",
			startIndex, len(logEntries))
		log.Panicf("LogEntries.getSlice: startIndex out of range. (%d < %d)", startIndex, 0)
	}
	if logEndIndex > len(logEntries)+1 {
		Debug(dError, "LogEntries.getSlice: endIndex out of range. endIndex: %d, len: %d.",
			endIndex, len(logEntries))
		log.Panicf("LogEntries.getSlice: endIndex out of range. (%d > %d)", endIndex, len(logEntries)+1+0)
	}
	if logStartIndex > logEndIndex {
		Debug(dError, "LogEntries.getSlice: startIndex > endIndex. (%d > %d)", startIndex, endIndex)
		log.Panicf("LogEntries.getSlice: startIndex > endIndex. (%d > %d)", startIndex, endIndex)
	}
	return append([]LogEntry(nil), logEntries[logStartIndex-1:logEndIndex-1]...)
}

func (rf *Raft) getBoundsWithTerm(term int) (minIndex int, maxIndex int) {
	if term == 0 {
		return 0, 0
	}
	minIndex = math.MaxInt
	maxIndex = -1

	entries := rf.log
	for i := 1; i <= len(entries); i++ {
		if entries.getEntry(i).Term == term {
			minIndex = Min(minIndex, i)
			maxIndex = Max(maxIndex, i)
		}
	}

	if maxIndex == -1 {
		return -1, -1
	}
	return
}
