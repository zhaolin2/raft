package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"6.824/labgob"
	"bytes"
	"log"
	//	"bytes"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type LogEntries []LogEntry

type LogEntry struct {
	Term    int         // Term number when created
	Command interface{} // Command to be excuted
}

type State int

const (
	FollowerState State = iota
	CandidateState
	LeaderState
)

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	applyCh        chan ApplyMsg
	applyCond      *sync.Cond   // used to wakeup applier goroutine after committing new entries
	replicatorCond []*sync.Cond // used to signal replicator goroutine to batch replicating entries
	state          State
	leaderId       int
	// Look at the paper's Figure 2 for a description of what
	//持久化信息 需要在回复rpc前更新存储
	currentTerm int        //服务器认为的最新任期
	votedFor    int        //当前任期下 该服务器投票给的节点
	log         LogEntries // 每个日志包含了一个命令以及被leader接收时候的任期号

	//保存的易失状态属性
	commitIndex int //已提交条目的最高下标 从0开始 递增
	lastApplied int // 应用在状态机的最高下标 从0开始 递增
	//状态 leader
	nextIndex  []int //对于每台服务器，将要发送的下一个日志条目的下标（初始化为 Leader 节点的最后一个日志条目下标 +1）
	matchIndex []int //每台服务器，已知的在服务器上复制的日志条目的最大下标（初始化为 0，单调递增）

	heartbeatTime time.Time
	electionTime  time.Time
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// Your code here (2A).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	return rf.currentTerm, rf.state == LeaderState
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)

	buffer := new(bytes.Buffer)
	encoder := labgob.NewEncoder(buffer)
	encoder.Encode(rf.currentTerm)

	bytes := buffer.Bytes()
	rf.persister.SaveRaftState(bytes)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }

	buffer := bytes.NewBuffer(data)
	decoder := labgob.NewDecoder(buffer)
	decoder.Decode(&rf.currentTerm)
}

// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {

	// Your code here (2D).

	return true
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	//index := -1
	//term := -1
	//isLeader := true

	// Your code here (2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.state != LeaderState {
		return -1, -1, false
	}

	logEntry := LogEntry{
		Command: command,
		Term:    rf.currentTerm,
	}
	rf.log = append(rf.log, logEntry)
	rf.info(dLog, "接收到command,command:%v", command)
	rf.sendEntries(false)

	return len(rf.log), rf.currentTerm, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.state = FollowerState
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 0)
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	for peer := range rf.peers {
		rf.nextIndex[peer] = 1
	}
	rf.applyCh = applyCh
	rf.leaderId = -1

	rf.setElectionTimeout(randHeartbeatTimeout())
	//rf.resetHeartBeatTimeOut()

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.info(dClient, "初始化成功")

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.applyLogsLoop()

	return rf
}

// index and term start from 1
func (logEntries LogEntries) getEntry(index int) *LogEntry {
	if index < 0 {
		log.Panic("LogEntries.getEntry: index < 0.\n")
	}
	if index == 0 {
		return &LogEntry{
			Command: nil,
			Term:    0,
		}
	}
	if index > len(logEntries) {
		return &LogEntry{
			Command: nil,
			Term:    -1,
		}
	}
	return &logEntries[index-1]
}

func (logEntries LogEntries) lastLogInfo() (index, term int) {
	index = len(logEntries)
	logEntry := logEntries.getEntry(index)
	return index, logEntry.Term
}

func (logEntries LogEntries) getSlice(startIndex, endIndex int) LogEntries {
	if startIndex <= 0 {
		Debug(dError, "LogEntries.getSlice: startIndex out of range. startIndex: %d, len: %d.",
			startIndex, len(logEntries))
		log.Panic("LogEntries.getSlice: startIndex out of range. \n")
	}
	if endIndex > len(logEntries)+1 {
		Debug(dError, "LogEntries.getSlice: endIndex out of range. endIndex: %d, len: %d.",
			endIndex, len(logEntries))
		log.Panic("LogEntries.getSlice: endIndex out of range.\n")
	}
	if startIndex > endIndex {
		Debug(dError, "LogEntries.getSlice: startIndex > endIndex. (%d > %d)", startIndex, endIndex)
		log.Panic("LogEntries.getSlice: startIndex > endIndex.\n")
	}
	return logEntries[startIndex-1 : endIndex-1]
}

func (rf *Raft) resetElectionTimeOut() {
	rf.info(dTimer, "重置选举时间")
	rf.setElectionTimeout(randElectionTimeout())
}

func (rf *Raft) resetHeartBeatTimeOut() {
	rf.info(dTimer, "重置心跳时间")
	rf.setHeartbeatTimeout(randHeartbeatTimeout())
}

// If RPC request or response contains term T > currentTerm: set currentTerm = T, convert to follower (§5.1)
// Check if current term is out of date when hearing from other peers,
// update term, revert to follower state and return true if necesarry
func (rf *Raft) checkTerm(term int, server int) bool {
	if rf.currentTerm < term {
		rf.info(dTerm, "准备成为follower,S%d Term is higher, updating term to T%d (%d > %d)",
			server, term, term, rf.currentTerm)
		rf.state = FollowerState
		rf.currentTerm = term
		rf.votedFor = -1
		rf.leaderId = -1
		return true
	}
	return false
}
