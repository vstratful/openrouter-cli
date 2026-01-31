package chat

import "strings"

// Command represents a chat command with autocomplete support.
type Command struct {
	Name        string
	Description string
}

// AvailableCommands returns all available chat commands.
func AvailableCommands() []Command {
	return []Command{
		{Name: CmdExit, Description: "Exit the application"},
		{Name: CmdModels, Description: "Change the AI model"},
		{Name: CmdQuit, Description: "Exit the application"},
		{Name: CmdResume, Description: "Resume a previous session"},
	}
}

// FilterCommands returns commands matching the given prefix.
func FilterCommands(prefix string) []Command {
	if prefix == "" || prefix[0] != '/' {
		return nil
	}
	all := AvailableCommands()
	if prefix == "/" {
		return all
	}
	var filtered []Command
	lowerPrefix := strings.ToLower(prefix)
	for _, cmd := range all {
		if strings.HasPrefix(strings.ToLower(cmd.Name), lowerPrefix) {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}
