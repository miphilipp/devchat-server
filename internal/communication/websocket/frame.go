package websocket

const (
	GetCommandMethod = iota + 1
	DeleteCommandMethod
	PostCommandMethod
	PatchCommandMethod
	NotifyCommandMethod
	ErrorCommandMethod
	HeartbeatCommandMethod
)

type RESTCommand struct {
	Ressource string `json:"ressource"`
	Method 	  int	 `json:"method"` 
}

type messageFrame struct {
	Command RESTCommand	`json:"command"`
	Source 	int			`json:"source"`
	ID 	    int			`json:"id"`
	Payload interface{}	`json:"payload"`
}

func newFrame(source, id int, command RESTCommand, payload interface{}) messageFrame  {
	return messageFrame{
		Command: command,
		Source: source,
		Payload: payload,
		ID: id,
	}
}

func makeErrorMessage(err error, id int) messageFrame {
	command := RESTCommand{
		Ressource: "",
		Method: ErrorCommandMethod,
	}

	return newFrame(-1, id, command, err)
}
