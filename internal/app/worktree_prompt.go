package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/session"
	"github.com/Gaurav-Gosain/tuios/internal/worktree"
)

// The rail's worktree dialog: a branch name asked over a repository, and a
// session made in a new worktree of it.
//
// Everything the dialog offers, the daemon already does: the new-worktree verb
// is the same one `tuios worktree new` runs, and the CLI is where the caller
// was before this dialog existed. What the dialog adds is the reach — a
// right-click on a session row or a repository's group header — and the two
// defaults the verb itself derives, the session name and the folder, shown
// where the name is typed so the user can see what enter will do.
//
// The folder is shown and not asked for. The verb takes no path; it puts every
// worktree it makes under $XDG_DATA_HOME/tuios/worktrees/<repo>/<branch> (see
// worktree.DefaultDir), and a dialog row that cannot change an answer is not a
// question, so the folder reads as the quiet line it is.

// worktreePromptState is the open dialog. The zero value is no dialog; the
// pointer is nil when it is closed, which WorktreePromptOpen reports.
type worktreePromptState struct {
	// RepoRoot is the main checkout the worktree branches from. It is captured
	// when the dialog opens, for the same reason filePromptState keeps its dir:
	// the cached listing can be replaced by a reply already in flight, and the
	// create must act on the repository the user was looking at.
	RepoRoot string
	// Input is the typed branch.
	Input string
	// NameInput is the typed session name, empty for the derived one. Tab
	// moves between this and the branch field.
	NameInput string
	// Field is the field the keyboard is in: worktreeFieldBranch or
	// worktreeFieldName.
	Field int
	// Err is the last refusal, drawn under the field so a second try can see
	// what was wrong with the first.
	Err string
}

// The two fields, as the tab key cycles them.
const (
	worktreeFieldBranch = iota
	worktreeFieldName
)

// WorktreePromptOpen reports whether the worktree dialog owns the keyboard.
func (m *OS) WorktreePromptOpen() bool { return m.worktreePrompt != nil }

// WorktreePromptRepoRoot is the repository the open dialog branches from, or
// "" when it is closed.
func (m *OS) WorktreePromptRepoRoot() string {
	if m.worktreePrompt == nil {
		return ""
	}
	return m.worktreePrompt.RepoRoot
}

// WorktreePromptBranch is the branch as typed, trimmed of the spaces the field
// would hand the daemon, or "" when the dialog is closed. It is the input
// package's read on the field, which cannot reach the unexported state.
func (m *OS) WorktreePromptBranch() string {
	if m.worktreePrompt == nil {
		return ""
	}
	return strings.TrimSpace(m.worktreePrompt.Input)
}

// WorktreePromptSessionName is the typed session name, trimmed, or "" when the
// dialog is closed or the field is, which is how the verb reads "derive it".
func (m *OS) WorktreePromptSessionName() string {
	if m.worktreePrompt == nil {
		return ""
	}
	return strings.TrimSpace(m.worktreePrompt.NameInput)
}

// BeginWorktreePromptForSession opens the dialog on the repository a session's
// worktree record names. A session with no record — an ordinary session, whose
// full directory the listing does not carry — is refused with a word, which is
// what the dimmed menu row says too: this release offers the dialog only where
// the repository is known. The refusal says what is missing and not "not a
// worktree", because a plain session can be in a repository's main checkout;
// what the rail lacks is the path to it.
func (m *OS) BeginWorktreePromptForSession(name string) {
	info := m.sessionWorktreeInfo(name)
	if info == nil {
		m.ShowNotification("The rail doesn't know which repository this session is in.", "info", m.Settings.NotificationDuration)
		return
	}
	m.openWorktreePrompt(info.RepoRoot)
}

// BeginWorktreePromptForRepo opens the dialog on a repository's group header.
// The header names the repository; the main checkout comes from any member
// session's worktree record, and there is one while the group is on screen at
// all.
func (m *OS) BeginWorktreePromptForRepo(repo string) {
	root := m.worktreeRepoRoot(repo)
	if root == "" {
		m.ShowNotification("No session in this repository right now.", "info", m.Settings.NotificationDuration)
		return
	}
	m.openWorktreePrompt(root)
}

// openWorktreePrompt captures the repository and opens on an empty branch.
func (m *OS) openWorktreePrompt(repoRoot string) {
	m.worktreePrompt = &worktreePromptState{RepoRoot: repoRoot}
}

// sessionWorktreeInfo is the worktree record the cached listing carries for a
// session, or nil. Reading the cache and not the socket is the rule every
// other rail surface keeps: this runs where a round trip would freeze the
// client.
func (m *OS) sessionWorktreeInfo(name string) *session.WorktreeInfo {
	if m.DaemonClient == nil || name == "" {
		return nil
	}
	return m.DaemonClient.SessionWorktree(name)
}

// worktreeRepoRoot finds the main checkout a repository's group header stands
// for, from the worktree record of any session still in it. Empty when no
// cached session names the repository, which for a row on screen means the
// listing changed since it was drawn.
func (m *OS) worktreeRepoRoot(repo string) string {
	if m.DaemonClient == nil || repo == "" {
		return ""
	}
	for _, s := range m.DaemonClient.CachedSessions() {
		if s.Worktree != nil && s.Worktree.Repo == repo && s.Worktree.RepoRoot != "" {
			return s.Worktree.RepoRoot
		}
	}
	return ""
}

// WorktreePromptNextField moves the keyboard between the branch and the
// session name, cycling, the way the confirmation dialogs cycle their rows.
func (m *OS) WorktreePromptNextField() {
	if m.worktreePrompt == nil {
		return
	}
	m.worktreePrompt.Field = (m.worktreePrompt.Field + 1) % 2
}

// worktreeFieldText is a field's text as typed, or "" when the dialog is
// closed.
func (m *OS) worktreeFieldText(field int) string {
	if m.worktreePrompt == nil {
		return ""
	}
	if field == worktreeFieldName {
		return m.worktreePrompt.NameInput
	}
	return m.worktreePrompt.Input
}

// WorktreePromptType adds typed text to the field the keyboard is in, laundered
// the way every name in this codebase is: only what the dialog will actually
// draw survives. A branch may hold a slash, which is why there is no
// file-prompt-style "trailing / means folder" rule here to fight.
func (m *OS) WorktreePromptType(text string) {
	if m.worktreePrompt == nil || text == "" {
		return
	}
	if m.worktreePrompt.Field == worktreeFieldName {
		m.worktreePrompt.NameInput += printableRunes(text)
	} else {
		m.worktreePrompt.Input += printableRunes(text)
	}
	m.worktreePrompt.Err = ""
}

// WorktreePromptBackspace drops the last rune of the active field, in runes,
// for the same reason RenameBackspace counts in runes.
func (m *OS) WorktreePromptBackspace() {
	if m.worktreePrompt == nil {
		return
	}
	if m.worktreePrompt.Field == worktreeFieldName {
		if m.worktreePrompt.NameInput == "" {
			return
		}
		m.worktreePrompt.NameInput = dropLastRune(m.worktreePrompt.NameInput)
	} else {
		if m.worktreePrompt.Input == "" {
			return
		}
		m.worktreePrompt.Input = dropLastRune(m.worktreePrompt.Input)
	}
	m.worktreePrompt.Err = ""
}

// WorktreePromptClearInput empties the active field.
func (m *OS) WorktreePromptClearInput() {
	if m.worktreePrompt == nil {
		return
	}
	if m.worktreePrompt.Field == worktreeFieldName {
		m.worktreePrompt.NameInput = ""
	} else {
		m.worktreePrompt.Input = ""
	}
}

// WorktreePromptCancel closes the dialog without running anything.
func (m *OS) WorktreePromptCancel() { m.worktreePrompt = nil }

// worktreePromptRepo is the repository's name as the dialog shows it: the base
// of the main checkout, which is what the rail's group header says too.
func (m *OS) worktreePromptRepo() string {
	if m.worktreePrompt == nil || m.worktreePrompt.RepoRoot == "" {
		return ""
	}
	return filepath.Base(m.worktreePrompt.RepoRoot)
}

// worktreePromptPath is the folder the worktree will land in, derived from the
// branch as it is typed, the way the daemon will derive it from the same
// branch. It runs no git and takes no validity opinion: this is a preview,
// drawn per keystroke, and the branch is judged once, on submit. An empty
// branch names the folder of the branch that is not there yet, so it reads
// empty.
func (m *OS) worktreePromptPath() string {
	if m.worktreePrompt == nil || strings.TrimSpace(m.worktreePrompt.Input) == "" {
		return ""
	}
	return worktree.PathFor(worktree.DefaultDir(), m.worktreePrompt.RepoRoot, strings.TrimSpace(m.worktreePrompt.Input))
}

// WorktreePromptSubmit checks the branch and hands the create to the daemon,
// in a command, because a create is a round trip that can take a clone's worth
// of minutes and this runs on the Update goroutine. The dialog closes when the
// work is handed off; the worktreeCreatedMsg handler says what became of it.
//
// The two checks here are the ones the user can fix without a round trip.
// The daemon owns the real refusals — a taken branch, a taken session name —
// and those come back as a notification.
func (m *OS) WorktreePromptSubmit() tea.Cmd {
	if m.worktreePrompt == nil {
		return nil
	}
	branch := strings.TrimSpace(m.worktreePrompt.Input)
	if branch == "" {
		m.worktreePrompt.Err = "A branch name is required."
		return nil
	}
	if err := worktree.ValidBranch(branch); err != nil {
		m.worktreePrompt.Err = err.Error()
		return nil
	}
	repoRoot := m.worktreePrompt.RepoRoot
	sessionName := strings.TrimSpace(m.worktreePrompt.NameInput)
	m.worktreePrompt = nil
	return m.createWorktreeCmd(repoRoot, branch, sessionName)
}

// worktreeVerbParams is the daemon call a create makes: the repository, the
// branch, and the session name when the user typed one — an empty name is how
// the verb says "derive it". It is its own function so a test can assert the
// call without running it.
func worktreeVerbParams(repoRoot, branch, sessionName string) map[string]any {
	params := map[string]any{"repo": repoRoot, "branch": branch}
	if sessionName != "" {
		params["name"] = sessionName
	}
	return params
}

// createWorktreeCmd runs the daemon's new-worktree verb, the same one
// `tuios worktree new` runs. The timeout is the CLI's own: a clone on another
// machine is bounded at four minutes there, and this waits a little longer.
func (m *OS) createWorktreeCmd(repoRoot, branch, sessionName string) tea.Cmd {
	dial := m.verbDialer()
	params := worktreeVerbParams(repoRoot, branch, sessionName)
	return func() tea.Msg {
		c, err := dial()
		if err != nil {
			return worktreeCreatedMsg{Err: err}
		}
		defer func() { _ = c.Close() }()

		raw, err := c.CallWithTimeout("new-worktree", params, 5*time.Minute)
		if err != nil {
			return worktreeCreatedMsg{Err: err}
		}
		var res struct {
			Session string `json:"session"`
		}
		if err := json.Unmarshal(raw, &res); err != nil {
			return worktreeCreatedMsg{Err: err}
		}
		return worktreeCreatedMsg{Session: res.Session}
	}
}

// worktreeCreatedMsg is a finished create on its way back to the loop.
type worktreeCreatedMsg struct {
	// Session is the session made in the new worktree, empty on failure.
	Session string
	// Err is what went wrong, nil on success.
	Err error
}

// handleWorktreeCreated says what became of a create: attach to the session it
// made, the way the CLI attaches at once, and say so; or say what failed. The
// switch touches the daemon client and the model, which is why it lives here
// and not in the command.
func (m *OS) handleWorktreeCreated(msg worktreeCreatedMsg) tea.Cmd {
	if msg.Err != nil {
		m.ShowNotification("Worktree failed: "+msg.Err.Error(), "error", m.Settings.NotificationDuration*2)
		return nil
	}
	if msg.Session == "" {
		return nil
	}
	m.ShowNotification("Created worktree session '"+msg.Session+"', attaching.", "info", m.Settings.NotificationDuration)
	if err := m.SwitchToSession(msg.Session); err != nil {
		m.ShowNotification("Could not attach to "+msg.Session+": "+err.Error(), "warning", m.Settings.NotificationDuration*2)
	}
	// The listing the rail reads is refreshed off this goroutine; the new
	// session joins its repository's group when it lands.
	if client := m.DaemonClient; client != nil {
		return func() tea.Msg {
			client.TryRefreshSessionList()
			return renameListingRefreshedMsg{}
		}
	}
	return nil
}
