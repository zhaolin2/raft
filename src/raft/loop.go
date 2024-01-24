package raft

import "time"

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
// 超时检查
func (rf *Raft) ticker() {
	for rf.killed() == false {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

		rf.mu.Lock()

		if rf.state == LeaderState && time.Now().After(rf.heartbeatTime) {
			rf.info(dTimer, "广播心跳.")
			rf.sendEntries(true)
		}

		if time.Now().After(rf.electionTime) {
			rf.info(dTimer, "转换为候选者，开始新的任期.")
			rf.raiseElection()
		}

		rf.mu.Unlock()

		time.Sleep(time.Duration(TickInterval) * time.Millisecond)

	}
}

func (rf *Raft) applyLogsLoop() {
	for !rf.killed() {
		rf.mu.Lock()
		appliedMsgs := []ApplyMsg{}
		for rf.commitIndex > rf.lastApplied {
			rf.lastApplied++
			appliedMsgs = append(appliedMsgs, ApplyMsg{
				CommandValid: true,
				Command:      rf.log.getEntry(rf.lastApplied).Command,
				CommandIndex: rf.lastApplied,
			})
			rf.info(dLog2, "apply日志到自己的状态机 lastApplied:%d,rf.commitIndex:%d command:%v", rf.lastApplied, rf.commitIndex, rf.log.getEntry(rf.lastApplied).Command)
		}
		rf.mu.Unlock()
		for _, msg := range appliedMsgs {
			rf.applyCh <- msg
		}
		time.Sleep(time.Duration(TickInterval) * time.Millisecond)
	}
}
