package raftkv

import (
	"log"
	"sync"
	"time"

	"github.com/Fallensouls/raft/labgob"
	"github.com/Fallensouls/raft/labrpc"
	"github.com/Fallensouls/raft/raft"
)

const Debug = 0

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Key       string
	Value     string
	Operation string
	ID        string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	done     chan int
	db       map[string]string // key-value database
	executed map[string]bool   // the set of operations which have been executed
}

func (kv *KVServer) Get(req *GetRequest, res *GetResponse) {
	// return if receiver isn't the leader.
	//log.Printf("state: %v", kv.rf.State())
	if kv.rf.State() != raft.Leader {
		res.WrongLeader = true
		return
	}

	// ensure that receiver is still the leader.
	kv.rf.Broadcast()
	if kv.rf.State() != raft.Leader {
		res.WrongLeader = true
		return
	}

	res.WrongLeader = false
	var ok bool
	kv.mu.Lock()
	if res.Value, ok = kv.db[req.Key]; ok {
		res.Err = OK
	} else {
		res.Err = ErrNoKey
	}
	kv.mu.Unlock()
	//log.Println(res)
}

func (kv *KVServer) PutAppend(req *PutAppendRequest, res *PutAppendResponse) {
	// return if receiver isn't the leader.
	if kv.rf.State() != raft.Leader {
		res.WrongLeader = true
		return
	}
	res.WrongLeader = false

	kv.mu.Lock()
	_, ok := kv.executed[req.ID]
	kv.mu.Unlock()
	if ok {
		res.Err = ErrExecuted
		return
	}

	index, _, _ := kv.rf.Start(Op{Key: req.Key, Value: req.Value, Operation: req.Op, ID: req.ID})
	timeout := make(chan struct{})
	go func() {
		time.Sleep(time.Second)
		timeout <- struct{}{}
		close(timeout)
	}()
	select {
	case doneIndex, _ := <-kv.done:
		if doneIndex == index {
			res.Err = OK
			return
		}
	case <-timeout:
		res.Err = ErrTimeout
		return
	}
}

func (kv *KVServer) apply() {
	for {
		select {
		case msg, ok := <-kv.applyCh:
			if ok && !msg.NoOpCommand {
				op := msg.Command.(Op)
				//log.Printf("value in server: %v", op.Value)
				kv.mu.Lock()
				switch op.Operation {
				case "Put":
					kv.db[op.Key] = op.Value
					kv.executed[op.ID] = true
				case "Append":
					kv.db[op.Key] += op.Value
					kv.executed[op.ID] = true
				default:
				}
				if kv.rf.State() == raft.Leader {
					kv.done <- msg.CommandIndex
				}
				kv.mu.Unlock()
			}
		}
	}
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *KVServer) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.
	kv.done = make(chan int, 50)
	kv.db = make(map[string]string)
	kv.executed = make(map[string]bool)
	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	go kv.apply()

	return kv
}