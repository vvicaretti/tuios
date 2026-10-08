package input

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Gaurav-Gosain/tuios/internal/app"
	"github.com/Gaurav-Gosain/tuios/internal/config"
	"github.com/Gaurav-Gosain/tuios/internal/session"
	"github.com/Gaurav-Gosain/tuios/internal/worktree"
)

// The worktree actions, dispatched. The menu rows are handed to the same
// dispatcher a keypress goes through, so the two carries — a session row's and
// a repository header's — are asserted here, where the dispatcher lives, and
// not against the menu builders' own entry points.
//
// The carry is set the way a menu row sets it: ContextMenuSelectedAction reads
// the menu's session field into the carry at the moment the row is chosen.

// menuOS is a client with a seeded listing and no socket, which is every
// daemon the carry tests need: the dialog reads the cache, and the create is
// never run here.
func menuOS(t *testing.T) *app.OS {
	t.Helper()
	o := app.NewOS(app.OSOptions{UserConfig: config.DefaultConfig(), KeybindRegistry: config.NewKeybindRegistry(config.DefaultConfig())})
	o.DaemonClient = &session.TUIClient{}
	o.DaemonClient.UpdateSessionCache([]session.SessionInfo{
		{Name: "main"},
		{Name: "tuios-feat-one", Worktree: &session.WorktreeInfo{
			Info: worktree.Info{Repo: "tuios", RepoRoot: "/src/tuios", Branch: "feat/one"},
		}},
		{Name: "docs-fix", Worktree: &session.WorktreeInfo{
			Info: worktree.Info{Repo: "docs", RepoRoot: "/src/docs", Branch: "fix/typo"},
		}},
	})
	return o
}

// chooseRow plants a one-row menu carrying the given session and takes its
// action the way a click or an enter does.
func chooseRow(t *testing.T, o *app.OS, carry, action string) {
	t.Helper()
	o.ContextMenu = &app.ContextMenu{
		SessionID: carry,
		Items:     []app.ContextMenuItem{{Action: action}},
		Selected:  0,
	}
	if got := o.ContextMenuSelectedAction(); got != action {
		t.Fatalf("the row chose %q, want %q", got, action)
	}
	o.CloseContextMenu()
}

// TestSessionNewWorktreeCarriesTheRow: the session row's worktree action opens
// the dialog on the session the menu was opened on, not on the attached one.
func TestSessionNewWorktreeCarriesTheRow(t *testing.T) {
	o := menuOS(t)
	chooseRow(t, o, "tuios-feat-one", "session_new_worktree")

	GetDispatcher().Dispatch("session_new_worktree", tea.KeyPressMsg{}, o)
	if !o.WorktreePromptOpen() || o.WorktreePromptRepoRoot() != "/src/tuios" {
		t.Fatalf("the action opened the dialog with the carry unset: open=%v", o.WorktreePromptOpen())
	}
}

// TestRepoNewWorktreeCarriesTheRow: the repository header's worktree action
// opens the dialog on the repository the carry names.
func TestRepoNewWorktreeCarriesTheRow(t *testing.T) {
	o := menuOS(t)
	chooseRow(t, o, "docs", "repo_new_worktree")

	GetDispatcher().Dispatch("repo_new_worktree", tea.KeyPressMsg{}, o)
	if !o.WorktreePromptOpen() || o.WorktreePromptRepoRoot() != "/src/docs" {
		t.Fatalf("the action opened the dialog with the carry unset: open=%v", o.WorktreePromptOpen())
	}
}

// TestTheWorktreeDialogTakesItsKeys: typed characters reach the field, tab
// moves to the session name, enter refuses to create on an empty one, and esc
// closes.
func TestTheWorktreeDialogTakesItsKeys(t *testing.T) {
	o := menuOS(t)
	chooseRow(t, o, "tuios-feat-one", "session_new_worktree")
	GetDispatcher().Dispatch("session_new_worktree", tea.KeyPressMsg{}, o)

	for _, r := range "feat/retry" {
		o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Text: string(r)}, o)
	}
	if got := o.WorktreePromptBranch(); got != "feat/retry" {
		t.Fatalf("the field holds %q, want what was typed", got)
	}

	// Tab reaches the session name, and what is typed there lands in it.
	o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Code: tea.KeyTab}, o)
	for _, r := range "my-name" {
		o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Text: string(r)}, o)
	}
	if got := o.WorktreePromptSessionName(); got != "my-name" {
		t.Fatalf("the name field holds %q, want what was typed", got)
	}

	// Enter on an empty branch is a refusal in the dialog, not a create. The
	// branch here is not empty, so clear it the way ctrl+u does first — which
	// means going back to the branch field, because ctrl+u clears the active
	// one.
	o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Code: tea.KeyTab}, o)
	o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, o)
	_, cmd := handleWorktreePromptInput(tea.KeyPressMsg{Code: tea.KeyEnter}, o)
	if cmd != nil {
		t.Error("an empty branch produced a create")
	}
	o, _ = handleWorktreePromptInput(tea.KeyPressMsg{Code: tea.KeyEsc}, o)
	if o.WorktreePromptOpen() {
		t.Fatal("esc did not close the dialog")
	}
}
