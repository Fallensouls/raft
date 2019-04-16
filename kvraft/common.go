package raftkv

const (
	OK          = "OK"
	ErrNoKey    = "ErrNoKey"
	ErrExecuted = "ErrExecuted"
	ErrTimeout  = "ErrTimeout"
)

type Err string

// Put or Append
type PutAppendRequest struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	ID string // id of the request
}

type PutAppendResponse struct {
	WrongLeader bool
	Err         Err
}

type GetRequest struct {
	Key string
	// You'll have to add definitions here.
	ID string // id of the request
}

type GetResponse struct {
	WrongLeader bool
	Err         Err
	Value       string
}
