package entity

import "os/exec"

type Command struct {
	Head      string
	Arguments []string
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

func NewCommand(head, category string, args []string) (*Command, error) {
	return &Command{
		Head:      head,
		Arguments: args,
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
func (c *Command) Run() (string, error) {
	cmd := exec.Command(c.Head, c.Arguments...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}
