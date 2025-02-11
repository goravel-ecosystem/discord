package commands

import (
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
)

type Heartbeat struct {
}

// Signature The name and signature of the console command.
func (receiver *Heartbeat) Signature() string {
	return "command:name"
}

// Description The console command description.
func (receiver *Heartbeat) Description() string {
	return "Command description"
}

// Extend The console command extend.
func (receiver *Heartbeat) Extend() command.Extend {
	return command.Extend{}
}

// Handle Execute the console command.
func (receiver *Heartbeat) Handle(ctx console.Context) error {
	
	return nil
}
