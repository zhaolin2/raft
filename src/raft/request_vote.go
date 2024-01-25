package raft

import "sync"

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int // candidate's term
	CandidateId  int // candidate requesting vote
	LastLogIndex int // index of candidate’s last log entry (§5.4)
	LastLogTerm  int // term of candidate’s last log entry (§5.4)
}

func (rf *Raft) tryBuilderRequestVote() *RequestVoteArgs {

	lastLogIndex, lastLogTerm := rf.log.lastLogInfo()
	args := &RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}

	return args
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

/*
*
增加任期
投票给自己
重制选举定时器
发送vote的rpc
*/
func (rf *Raft) raiseElection() {
	rf.state = CandidateState
	rf.currentTerm++
	rf.info(dTerm, "开始新的任期")
	rf.votedFor = rf.me
	rf.persist()
	rf.resetElectionTimeOut()

	args := rf.tryBuilderRequestVote()

	voteCount := 1
	var once sync.Once

	//发送投票
	for server := range rf.peers {
		if server == rf.me {
			continue
		} else {
			rf.info(dVote, "开始发送投票请求,发送给S%d", server)
			go rf.candidateRequestVote(&voteCount, args, &once, server)
		}
	}

}

func (rf *Raft) candidateRequestVote(voteCount *int, args *RequestVoteArgs, once *sync.Once, server int) {
	reply := &RequestVoteReply{}
	result := rf.sendRequestVote(server, args, reply)

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if result {
		rf.info(dVote, "接收到投票 S%d <- S%d T%d", rf.me, server, rf.currentTerm)

		if rf.currentTerm < reply.Term {
			rf.info(dVote, "replay中的任期更大 单次投票无效 (S%d  T%d > T%d)", rf.me, reply.Term, rf.currentTerm)
			return
		}

		if rf.currentTerm != args.Term {
			rf.info(dWarn, "当前任期已跟传入任期不同,投票作废 (S%d currentTerm:%d requestTerm:%d)", rf.me, rf.currentTerm, args.Term)
		}

		rf.checkTerm(reply.Term, server)

		if reply.VoteGranted {
			*voteCount++
			rf.info(dVote, "接收到投票 S%d <- S%d  T%d", rf.me, server, rf.currentTerm)

			if *voteCount > len(rf.peers)/2 {
				once.Do(func() {
					rf.info(dLeader, "成为Leader")
					rf.state = LeaderState
					index, _ := rf.log.lastLogInfo()
					for i := range rf.peers {
						rf.nextIndex[i] = index + 1
						rf.matchIndex[i] = 0
					}
					rf.sendEntries(true)
				})
			}
		}

	} else {
		rf.info(dVote, "接收投票结果失败 S%d <- S%d T%d", rf.me, server, rf.currentTerm)
	}

}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	rf.info(dVote, "接收到请求, voteFrom:S%d,term:T%d", args.CandidateId, args.Term)

	if rf.currentTerm > args.Term {
		rf.info(dVote, "拒绝投票,当前任期更高 (T%d > T%d)", rf.currentTerm, args.Term)
		reply.Term = rf.currentTerm
		return
	}

	rf.checkTerm(args.Term, args.CandidateId)
	rf.currentTerm = args.Term

	index, term := rf.log.lastLogInfo()

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		if term < args.LastLogTerm || (term == args.LastLogTerm && index <= args.LastLogIndex) {
			rf.info(dVote, "开始投票,投票给 S%d at T%d ", args.CandidateId, args.Term)
			rf.info(dLog2, "打印判断条件, (T%d < T%d || (T%d==T%d && %d <= %d))", term, args.LastLogTerm, term, args.LastLogTerm, index, args.LastLogIndex)
			reply.VoteGranted = true
			rf.votedFor = args.CandidateId
			rf.persist()
			rf.resetElectionTimeOut()
			return
		}
		rf.info(dVote, "当前投票不符合条件,不进行投票,(args.index: %d,args.term: %d)", args.LastLogIndex, args.LastLogTerm)

	}
	rf.info(dVote, "已投票,拒绝投票")

}
