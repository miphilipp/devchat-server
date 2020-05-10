package websocket

import "context"

const (
	GetCommandMethod = iota + 1
	DeleteCommandMethod
	PostCommandMethod
	PatchCommandMethod
	NotifyCommandMethod
	ErrorCommandMethod
	HeartbeatCommandMethod
)

const (
	RequestContextCommandKey = "command"
	RequestContextSourceKey  = "Source"
	RequestContextIDKey      = "id"
)

type RESTCommand struct {
	Ressource string `json:"ressource"`
	Method    int    `json:"method"`
}

type messageFrame struct {
	Command RESTCommand `json:"command"`
	Source  int         `json:"source"`
	ID      int         `json:"id"`
	Payload interface{} `json:"payload"`
}

func NewRequestContext(command RESTCommand, id, sourceCtx int) context.Context {
	ctx := context.WithValue(context.Background(), RequestContextCommandKey, command)
	ctx = context.WithValue(ctx, RequestContextSourceKey, sourceCtx)
	return context.WithValue(ctx, RequestContextIDKey, id)
}

func newFrame(source, id int, command RESTCommand, payload interface{}) messageFrame {
	return messageFrame{
		Command: command,
		Source:  source,
		Payload: payload,
		ID:      id,
	}
}

func makeErrorMessage(err error, id int, ressource string) messageFrame {
	command := RESTCommand{
		Ressource: ressource,
		Method:    ErrorCommandMethod,
	}

	return newFrame(-1, id, command, err)
}
