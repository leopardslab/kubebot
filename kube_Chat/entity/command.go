package entity

type Command struct {
	CmdStr    string
	Head      string
	Arguments []string
	Flags     []string
	ArgCount  int
	Category  string
}

type CommandResult struct {
	Cmd    string
	Result string
	Error  string // error from the exec
}

type CommandContext struct {
	Query      []string //Required arguments for particular command
	Command    Command
	IsRequired bool // False if all arguments are satisfied
}

func NewCommand(cmd, head, category string, args, flags []string, argc int) (*Command, error) {
	return &Command{
		CmdStr:    cmd,
		Head:      head,
		Arguments: args,
		Flags:     flags,
		ArgCount:  argc,
		Category:  category,
	}, nil
}

func NewCommandResult(cmd, result, error string) *CommandResult {
	return &CommandResult{
		Cmd:    cmd,
		Result: result,
		Error:  error,
	}
}

func NewCommandContext(com Command, query []string, req bool) *CommandContext {
	return &CommandContext{
		Command:    com,
		Query:      query,
		IsRequired: req,
	}
}
