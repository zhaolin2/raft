package main

import (
	"log"
	"math/rand"
	"time"
)

const BaseHeartbeatTimeout int64 = 300 // Lower bound of heartbeat timeout. Election is raised when timeout as a follower.
const BaseElectionTimeout int64 = 1000 // Lower bound of election timeout. Another election is raised when timeout as a candidate.

const RandomFactor float64 = 0.8 // Factor to control upper bound of heartbeat timeouts and election timeouts.

func randElectionTimeout() time.Duration {
	extraTime := int64(float64(rand.Int63()%BaseElectionTimeout) * RandomFactor)
	return time.Duration(extraTime+BaseElectionTimeout) * time.Millisecond
}

func randHeartbeatTimeout() time.Duration {
	extraTime := int64(float64(rand.Int63()%BaseHeartbeatTimeout) * RandomFactor)
	return time.Duration(extraTime+BaseHeartbeatTimeout) * time.Millisecond
}

func main() {
	log.Printf("%s", time.Duration(30)*time.Millisecond)

}
