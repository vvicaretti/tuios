package input

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/app"
)

// handleWorktreePromptInput drives the rail's worktree dialog.
//
// It is the file name prompt's keyboard with none of the confirmations: one
// text field, enter creates, esc cancels. There is no yes or no to pick, so
// the keys are the field's keys and nothing else.
func handleWorktreePromptInput(msg tea.KeyPressMsg, o *app.OS) (*app.OS, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		o.WorktreePromptCancel()
	case "enter":
		return o, o.WorktreePromptSubmit()
	case "tab":
		o.WorktreePromptNextField()
	case "backspace":
		o.WorktreePromptBackspace()
	case "ctrl+u":
		o.WorktreePromptClearInput()
	case "space":
		o.WorktreePromptType(" ")
	default:
		typePromptText(msg, o.WorktreePromptType)
	}
	return o, nil
}
