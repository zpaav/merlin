/*
Merlin is a post-exploitation command and control framework.

This file is part of Merlin.
Copyright (C) 2023  Russel Van Tuyl

Merlin is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
any later version.

Merlin is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Merlin.  If not, see <http://www.gnu.org/licenses/>.
*/

package ls

import (
	// Standard
	"fmt"
	"strings"
	"time"

	// 3rd Party
	"github.com/chzyer/readline"
	"github.com/mattn/go-shellwords"
	uuid "github.com/satori/go.uuid"

	// Internal
	agentAPI "github.com/Ne0nd0g/merlin/pkg/api/agents"
	"github.com/Ne0nd0g/merlin/pkg/api/messages"
	"github.com/Ne0nd0g/merlin/pkg/cli/commands"
	"github.com/Ne0nd0g/merlin/pkg/cli/core"
	"github.com/Ne0nd0g/merlin/pkg/cli/entity/help"
	"github.com/Ne0nd0g/merlin/pkg/cli/entity/menu"
	"github.com/Ne0nd0g/merlin/pkg/cli/entity/os"
)

// Command is an aggregate structure for a command executed on the command line interface
type Command struct {
	name   string      // name is the name of the command
	help   help.Help   // help is the Help structure for the command
	menus  []menu.Menu // menu is the Menu the command can be used in
	native bool        // native is true if the command is executed by an Agent using only Golang native code
	os     os.OS       // os is the supported operating system the Agent command can be executed on
}

// NewCommand is a factory that builds and returns a Command structure that implements the Command interface
func NewCommand() *Command {
	var cmd Command
	cmd.name = "ls"
	cmd.menus = []menu.Menu{menu.AGENT}
	cmd.os = os.ALL
	description := "List the files and folders of the provided filepath"
	// Style guide for usage https://developers.google.com/style/code-syntax
	usage := "ls [filePath]"
	example := "Merlin[agent][c1090dbc-f2f7-4d90-a241-86e0c0217786]» ls /var\n" +
		"\t[-]Created job eNJKIiLXXH for agent c1090dbc-f2f7-4d90-a241-86e0c0217786\n" +
		"\t[+]Results for job eNJKIiLXXH\n" +
		"\tDirectory listing for: /var\n\n" +
		"\tdrwxr-xr-x      2019-02-06 00:05:17     4096    backups\n" +
		"\tdrwxr-xr-x      2018-12-24 14:40:14     4096    cache\n" +
		"\tdgtrwxrwxrwx    2019-02-06 00:05:16     4096    crash\n" +
		"\tdrwxr-xr-x      2019-01-17 21:24:30     4096    lib\n" +
		"\tdgrwxrwxr-x     2018-04-24 04:34:22     4096    local\n" +
		"\tLrwxrwxrwx      2018-11-07 21:33:01     9       lock\n" +
		"\tdrwxrwxr-x      2019-02-06 00:05:39     4096    log\n" +
		"\tdgrwxrwxr-x     2018-07-24 23:03:56     4096    mail\n" +
		"\tdgtrwxrwxrwx    2018-07-24 23:09:50     4096    metrics\n" +
		"\tdrwxr-xr-x      2018-07-24 23:03:56     4096    opt\n" +
		"\tLrwxrwxrwx      2018-11-07 21:33:01     4       run\n" +
		"\tdrwxr-xr-x      2018-11-07 21:45:43     4096    snap\n" +
		"\tdrwxr-xr-x      2018-11-07 21:38:04     4096    spool\n" +
		"\tdtrwxrwxrwx     2019-02-06 00:05:38     4096    tmp"
	notes := "This command will not execute the ls or dir binary programs found on their associated host " +
		"operating systems. If a directory is not specified, Merlin will list the contents of the current working " +
		"directory. When specifying a Windows path, you must escape the backslash (e.g.,. C:\\Temp). " +
		"Wrap file paths containing a space in quotations. " +
		"Alternatively, Linux file paths with a space can be called without quotes by escaping the space " +
		"(e.g.,. /root/some\\ folder/). Relative paths can be used (e.g.,. ./../ or downloads\\\\Merlin) and they are " +
		"resolved to their absolute path."
	cmd.help = help.NewHelp(description, example, notes, usage)
	return &cmd
}

// Completer returns the data that is displayed in the CLI for tab completion depending on the menu the command is for
// Errors are not returned to ensure the CLI is not interrupted.
// Errors are logged and can be viewed by enabling debug output in the CLI
func (c *Command) Completer(m menu.Menu, id uuid.UUID) readline.PrefixCompleterInterface {
	if core.Debug {
		core.MessageChannel <- messages.UserMessage{
			Level:   messages.Debug,
			Message: fmt.Sprintf("entering into Completer() for the '%s' command with Menu: %s, and id: %s", c, m, id),
			Time:    time.Now().UTC(),
		}
	}
	return readline.PcItem(c.name)
}

// Do executes the command and returns a Response to the caller to facilitate changes in the CLI service
// m, an optional parameter, is the Menu the command was executed from
// id, an optional parameter, used to identify a specific Agent or Listener
// arguments, and optional, parameter, is the full unparsed string entered on the command line to include the
// command itself passed into command for processing
func (c *Command) Do(m menu.Menu, id uuid.UUID, arguments string) (response commands.Response) {
	if core.Debug {
		core.MessageChannel <- messages.UserMessage{
			Level:   messages.Debug,
			Message: fmt.Sprintf("entering into Do() for the '%s' command with Menu: %s, id: %s, and arguments: %s", c, m, id, arguments),
			Time:    time.Now().UTC(),
		}
	}

	// Parse the arguments
	args, err := shellwords.Parse(arguments)
	if err != nil {
		response.Message = &messages.UserMessage{
			Level:   messages.Warn,
			Error:   true,
			Message: fmt.Sprintf("there was an error parsing the arguments: %s", err),
			Time:    time.Now().UTC(),
		}
		err = nil
		return
	}

	// Check for help first
	if len(args) > 1 {
		switch strings.ToLower(args[1]) {
		case "help", "-h", "--help", "?", "/?":
			response.Message = &messages.UserMessage{
				Level:   messages.Info,
				Message: fmt.Sprintf("'%s' command help\n\nDescription:\n\t%s\nUsage:\n\t%s\nExample:\n\t%s\nNotes:\n\t%s", c, c.help.Description(), c.help.Usage(), c.help.Example(), c.help.Notes()),
				Time:    time.Now().UTC(),
			}
			return
		}
	} else {
		args = append(args, ".")
	}
	msg := agentAPI.LS(id, args)
	response.Message = &msg
	return
}

// Help returns a help.Help structure that can be used to view a command's Description, Notes, Usage, and an example
func (c *Command) Help(m menu.Menu) help.Help {
	if core.Debug {
		core.MessageChannel <- messages.UserMessage{
			Level:   messages.Debug,
			Message: fmt.Sprintf("entering into Help() for the '%s' command with Menu: %s", c, m),
			Time:    time.Now().UTC(),
		}
	}
	return c.help
}

// Menu checks to see if the command is supported for the provided menu
func (c *Command) Menu(m menu.Menu) bool {
	for _, v := range c.menus {
		if v == m || v == menu.ALLMENUS {
			return true
		}
	}
	return false
}

// OS returns the supported operating system the Agent command can be executed on
func (c *Command) OS() os.OS {
	return c.os
}

// String returns the unique name of the command as a string
func (c *Command) String() string {
	return c.name
}
