package app

import (
	"image/color"
	"strings"

	"github.com/Gaurav-Gosain/tuios/internal/overlay"
	"github.com/Gaurav-Gosain/tuios/internal/theme"
	"github.com/Gaurav-Gosain/tuios/internal/worktree"
)

// The worktree dialog's drawing.
//
// It is the file create prompt's shape with a different question in it: a
// centred overlay.Dialog, a line above the fields saying what repository the
// branch comes off, and quiet lines under them saying what enter will make.
// The reasoning is the file prompt's and is not repeated here; see
// render_file_prompt.go for why this is an overlay and not a rail row.
//
// Two fields, because a worktree names two things: the branch it checks out,
// and the session it is addressed by. The session field starts empty and its
// empty state shows the name the daemon will derive, so the default is what
// the field reads as rather than something the user has to imagine.

// worktreeDialogWidth is the preferred inner width. Wide enough for the
// derived folder, which is the longest line the dialog draws, cut from the
// front like every other path in these dialogs so the tail survives.
const worktreeDialogWidth = 54

// worktreeFieldLabelW is the label column both field rows share.
const worktreeFieldLabelW = 9

// renderWorktreePrompt draws the worktree dialog.
func (m *OS) renderWorktreePrompt() (string, overlay.Geometry, []overlayRowHit) {
	pal := theme.UI()
	bg := pal.Canvas
	width := overlay.DialogFitWidth(worktreeDialogWidth, m.GetRenderWidth())

	var body []string
	body = append(body, fileDialogLine(m.worktreePromptContext(width), width, pal.FgDim, bg))
	body = append(body,
		m.worktreePromptField(worktreeFieldBranch, "Branch", width, pal, bg),
		m.worktreePromptField(worktreeFieldName, "Session", width, pal, bg))

	// Under the fields: the folder the worktree will land in. It is derived
	// from the branch as it is typed, so it reads as the default it is. One
	// line under that, and it is the refusal when there is one, standing where
	// the folder line stood — a second try looks at the same place for what
	// went wrong.
	if path := m.worktreePromptPath(); path != "" {
		body = append(body, fileDialogLine("In "+truncPathLeft(shortenHome(path), max(width-4, 1)), width, pal.FgMute, bg))
	}
	note, ink := m.worktreePromptNote(), pal.FgMute
	if m.worktreePrompt.Err != "" {
		note, ink = m.worktreePrompt.Err, pal.Warn
	}
	body = append(body, fileDialogLine(note, width, ink, bg))

	content, geo := overlay.Dialog{
		Title: "new worktree",
		Width: width,
		Body:  strings.Join(body, "\n"),
		Hints: []overlay.Hint{
			{Key: overlay.EnterKey(), Label: "create"},
			{Key: "tab", Label: "next field"},
			{Key: "esc", Label: "cancel"},
		},
	}.Render(pal)
	return content, geo, nil
}

// worktreePromptField draws one field row: its label, the sigil on the field
// the keyboard is in, the text windows to its tail so what you are typing
// stays under the cursor, and the cursor on the active field only. A session
// field left empty shows the name the daemon will derive, in the quiet ink a
// default reads as.
func (m *OS) worktreePromptField(field int, label string, width int, pal overlay.Palette, bg color.Color) string {
	active := m.worktreePrompt.Field == field
	value := m.worktreeFieldText(field)

	// The empty session field reads as the derived name it will fall back to.
	quiet := false
	if value == "" && field == worktreeFieldName {
		if branch := m.WorktreePromptBranch(); branch != "" {
			value = worktree.SessionName(m.worktreePromptRepo(), branch)
			quiet = true
		}
	}

	marker, ink := " ", pal.Fg
	if active {
		marker, ink = overlay.Sigil(), pal.Fg
	} else {
		ink = pal.FgDim
	}
	if quiet {
		ink = pal.FgMute
	}

	text := renameFieldText(printableRunes(value), max(width-4-worktreeFieldLabelW, 1))
	row := overlay.Style(bg).Render(" ") +
		overlay.Style(bg).Foreground(pal.AccentBright).Bold(true).Render(marker) +
		overlay.Style(bg).Foreground(pal.FgMute).Render(labelPad(label)) +
		overlay.Style(bg).Foreground(ink).Render(text)
	if active {
		row += overlay.Cursor(" ", bg, pal.Fg)
	}
	return overlay.Fill(row, width, bg)
}

// labelPad pads a field label to the shared label column.
func labelPad(label string) string {
	return label + strings.Repeat(" ", max(worktreeFieldLabelW-len(label)-1, 1))
}

// worktreePromptContext is the line above the fields: which repository the
// worktree branches from. The main checkout's path is cut from the front like
// every other path in these dialogs, so the tail — the name the rail's group
// header shows — is the part that survives.
func (m *OS) worktreePromptContext(width int) string {
	root := truncPathLeft(shortenHome(m.worktreePrompt.RepoRoot), max(width-16, 1))
	return "Worktree of " + printableTitle(m.worktreePromptRepo()) + " (" + root + ")"
}

// worktreePromptNote is the standing hint under the fields, shown while it
// says nothing else.
func (m *OS) worktreePromptNote() string {
	return "The branch the worktree checks out."
}
