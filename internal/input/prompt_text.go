package input

import (
	tea "charm.land/bubbletea/v2"
)

// typePromptText funnels a keypress into a text prompt's field, the way every
// rail prompt takes text: the terminal's composed text when the event carries
// one, a single printable byte when it does not. It reports whether the key
// was text, so a caller's default branch is one call.
func typePromptText(msg tea.KeyPressMsg, typeInto func(string)) bool {
	key := msg.String()
	switch {
	case msg.Text != "":
		typeInto(msg.Text)
	case len(key) == 1 && key[0] >= 32 && key[0] <= 126:
		typeInto(key)
	default:
		return false
	}
	return true
}
