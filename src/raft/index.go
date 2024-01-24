package raft

import "log"

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
