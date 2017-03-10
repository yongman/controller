package controller

type Result interface{}

type CommandType int

const (
	REGION_COMMAND CommandType = iota
	CLUSTER_COMMAND
	MUTEX_COMMAND
	NOMUTEX_COMMAND
)

type Command interface {
	Type() CommandType
	Mutex() CommandType
	Execute(*Controller) (Result, error)
}
