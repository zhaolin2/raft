package raft

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

//const Debug = true
//
//func DPrintf(format string, a ...interface{}) (n int, err error) {
//	if Debug {
//		log.Printf(format, a...)
//	}
//	return
//}

type logTopic string

const (
	dClient  logTopic = "CLNT"
	dCommit  logTopic = "CMIT"
	dDrop    logTopic = "DROP"
	dError   logTopic = "ERRO"
	dInfo    logTopic = "INFO"
	dLeader  logTopic = "LEAD"
	dLog     logTopic = "LOG1"
	dLog2    logTopic = "LOG2"
	dPersist logTopic = "PERS"
	dSnap    logTopic = "SNAP"
	dTerm    logTopic = "TERM"
	dTest    logTopic = "TEST"
	dTimer   logTopic = "TIMR"
	dTrace   logTopic = "TRCE"
	dVote    logTopic = "VOTE"
	dWarn    logTopic = "WARN"
)

var debugStart time.Time
var debugVerbosity int

func init() {
	debugVerbosity = getVerbosity()
	debugStart = time.Now()

	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func Debug(topic logTopic, format string, a ...interface{}) {
	if debugVerbosity >= 1 {
		microseconds := time.Since(debugStart).Microseconds()
		microseconds /= 100
		prefix := fmt.Sprintf("%06d %v ", microseconds, string(topic))
		format = prefix + format
		log.Printf(format, a...)
	}
}

func (rf *Raft) info(topic logTopic, format string, a ...interface{}) {
	if debugVerbosity >= 1 {
		microseconds := time.Since(debugStart).Microseconds()
		microseconds /= 100
		//lastLogIndex, lastLogTerm := rf.log.lastLogInfo()

		prefix := fmt.Sprintf("%06d %v [S%d T%d commitIndex:%d,ApplyIndex:%d] ", microseconds, string(topic), rf.me, rf.currentTerm, rf.commitIndex, rf.lastApplied)

		format = prefix + format
		log.Printf(format, a...)
	}
}

// Retrieve the verbosity level from an environment variable
func getVerbosity() int {
	v := os.Getenv("VERBOSE")
	level := 0
	if v != "" {
		var err error
		level, err = strconv.Atoi(v)
		if err != nil {
			log.Fatalf("Invalid verbosity %v", v)
		}
	}
	return level
}

func Client(format string, a ...interface{}) {
	Debug(dClient, format, a)
}

func Min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func Max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
