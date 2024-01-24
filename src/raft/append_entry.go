package raft

import "time"

type AppendEntriesArgs struct {
	Term         int        // leader’s term
	LeaderId     int        // with leaderId follower can redirect clients
	PrevLogIndex int        // index of log entry immediately preceding new ones
	PrevLogTerm  int        // term of prevLogIndex entry
	Entries      []LogEntry // log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int        // leader's commitIndex
}

func (rf *Raft) tryBuildAppendEntriesArgs(peer int) *AppendEntriesArgs {

	nextIndex := rf.nextIndex[peer]

	args := &AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:     rf.me,
		PrevLogIndex: nextIndex - 1,
		PrevLogTerm:  rf.log.getEntry(nextIndex - 1).Term,
		LeaderCommit: rf.commitIndex,
	}

	return args
}

type AppendEntriesReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if len(args.Entries) == 0 {
		rf.info(dLog2, "接收到心跳 <- S%d", args.LeaderId)
	} else {
		rf.info(dLog2, "接收到日志 S%d <- S%d Received append entries at T%d.", rf.me, args.LeaderId, rf.currentTerm)
	}
	reply.Success = false

	//5.1实现
	if args.Term < rf.currentTerm {
		rf.info(dLog2, "接收到的日志的term小于当前任期,拒绝日志 (T%d > T%d)", rf.currentTerm, args.Term)
		reply.Term = rf.currentTerm
		return
	}

	if rf.currentTerm == args.Term && rf.state == CandidateState {
		rf.info(dLog2, "转变为follower")
		rf.state = FollowerState
	}

	rf.checkTerm(args.Term, args.LeaderId)
	reply.Term = rf.currentTerm
	rf.setElectionTimeout(randHeartbeatTimeout())
	rf.leaderId = args.LeaderId

	//5.2实现
	if args.PrevLogTerm == -1 && args.PrevLogTerm != rf.log.getEntry(args.PrevLogIndex).Term {
		rf.info(dLog2, "检查上一条日志的term失败,需要leader重试")
		return
	}

	for index, entry := range args.Entries {
		if rf.log.getEntry(index+1+args.PrevLogIndex).Term != entry.Term {
			//rf.log = append(rf.log.getSlice(1, index+1+args.PrevLogIndex), args.Entries[index:]...)
			rf.log = append(rf.getSlice(1, index+1+args.PrevLogIndex), args.Entries[index:]...)
			break
		}
	}

	rf.info(dLog2, "增加日志成功,logs:%v", args.Entries)

	if args.LeaderCommit > rf.commitIndex {
		rf.info(dCommit, "leader的commit更高 (%d > %d)", args.LeaderCommit, rf.commitIndex)
		rf.commitIndex = Min(args.LeaderCommit, args.PrevLogIndex+len(args.Entries))
		rf.info(dCommit, "更新commitIndex,index:%d", rf.commitIndex)
	}

	reply.Success = true
}

func (rf *Raft) sendEntries(isHeartbeat bool) {

	rf.setElectionTimeout(randHeartbeatTimeout())
	rf.setHeartbeatTimeout(time.Duration(HeartbeatInterval) * time.Millisecond)

	lastLogIndex, _ := rf.log.lastLogInfo()
	for peer := range rf.peers {
		if peer == rf.me {
			continue
		}
		nextIndex := rf.nextIndex[peer]
		args := rf.tryBuildAppendEntriesArgs(peer)

		if lastLogIndex >= nextIndex {
			// If last log index ≥ nextIndex for a follower:
			args.Entries = rf.getSlice(nextIndex, lastLogIndex+1)
			rf.info(dLog, "发送日志请求, -> S%d,[PreLogIndex:%d,PreLogTerm:%d,LeaderCommit:%d,logs:%v]", peer,
				args.PrevLogIndex, args.PrevLogTerm, args.LeaderCommit, args.Entries)
			//Debug(dLog, "S%d -> S%d Sending append entries at T%d. PLI: %d, PLT: %d, LC: %d. Entries: %v.",
			//	rf.me, peer, rf.currentTerm, args.PrevLogIndex,
			//	args.PrevLogTerm, args.LeaderCommit, args.Entries,
			//)
			go rf.leaderSendEntries(args, peer)
		} else if isHeartbeat {
			args.Entries = make([]LogEntry, 0)
			rf.info(dLog, "发送心跳请求 -> S%d", peer)
			go rf.leaderSendEntries(args, peer)
		}
	}
}

func (rf *Raft) leaderSendEntries(args *AppendEntriesArgs, server int) {
	reply := &AppendEntriesReply{}
	ok := rf.sendAppendEntries(server, args, reply)
	if ok {
		rf.mu.Lock()
		defer rf.mu.Unlock()
		rf.info(dLog, "接收到回复 S%d <- S%d", rf.me, server)
		if reply.Term < rf.currentTerm {
			rf.info(dLog, "当前节点的term更高,忽略响应 (%d > %d)", rf.currentTerm, reply.Term)
			return
		}
		if rf.currentTerm != args.Term {
			rf.info(dWarn, "当前节点的任期已改变,忽略响应 (%d -> %d)", args.Term, rf.currentTerm)
			return
		}
		if rf.checkTerm(reply.Term, server) {
			return
		}
		// If successful: update nextIndex and matchIndex for follower (§5.3)
		if reply.Success {
			rf.info(dLog, "接收日志成功 <- S%d", server)
			newNext := args.PrevLogIndex + 1 + len(args.Entries)
			newMatch := args.PrevLogIndex + len(args.Entries)
			rf.nextIndex[server] = maxInt(newNext, rf.nextIndex[server])
			rf.matchIndex[server] = maxInt(newMatch, rf.matchIndex[server])

			for N := len(rf.log); N > rf.commitIndex && rf.log.getEntry(N).Term == rf.currentTerm; N-- {
				count := 1
				for peer, matchIndex := range rf.matchIndex {
					if peer == rf.me {
						continue
					}
					if matchIndex >= N {
						count++
					}
				}
				if count > len(rf.peers)/2 {
					rf.commitIndex = N
					rf.info(dCommit, "日志获得大部分同意,提交. commitIndex:%d", rf.commitIndex)
					break
				}
			}

			return
		}

		// If AppendEntries fails because of log inconsistency: decrement nextIndex and retry (§5.3)
		if rf.nextIndex[server] > 1 {
			rf.nextIndex[server]--
		}
		lastLogIndex, _ := rf.log.lastLogInfo()
		nextIndex := rf.nextIndex[server]
		if lastLogIndex >= nextIndex {
			rf.info(dLog, "S%d <- S%d 请求被拒绝, 重试.", rf.me, server)
			entries := make([]LogEntry, lastLogIndex-nextIndex+1)
			copy(entries, rf.log.getSlice(nextIndex, lastLogIndex+1))
			newArg := &AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: nextIndex - 1,
				PrevLogTerm:  rf.log.getEntry(nextIndex - 1).Term,
				Entries:      entries,
			}
			go rf.leaderSendEntries(newArg, server)
		}

	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
