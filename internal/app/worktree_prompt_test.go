package app

import (
	"strings"
	"testing"

	"github.com/Gaurav-Gosain/tuios/internal/session"
	"github.com/Gaurav-Gosain/tuios/internal/sessiontree"
	"github.com/Gaurav-Gosain/tuios/internal/worktree"
)

// worktreePromptOS is the rail of worktreeRailOS with the cached listing seeded
// to say where each repository's main checkout is. The dialog reads the cache
// and not the socket, so a seeded cache is the whole daemon it needs; "main"
// has no worktree record, which is the case the menu dims and the dialog
// refuses.
func worktreePromptOS(t *testing.T) (*OS, sessiontree.Tree) {
	t.Helper()
	m, tree := worktreeRailOS(t, 120, 30)
	m.DaemonClient.UpdateSessionCache([]session.SessionInfo{
		{Name: "main"},
		{Name: "tuios-feat-one", Worktree: &session.WorktreeInfo{
			Info: worktree.Info{Repo: "tuios", RepoRoot: "/src/tuios", Branch: "feat/one"},
		}},
		{Name: "docs-fix", Worktree: &session.WorktreeInfo{
			Info: worktree.Info{Repo: "docs", RepoRoot: "/src/docs", Branch: "fix/typo"},
		}},
	})
	return m, tree
}

// TestSessionMenuOffersANewWorktreeWhereTheRepoIsKnown: the session row's menu
// offers a worktree of the session's own repository, live on a worktree
// session and dimmed on a plain one. A dimmed row says the action exists;
// dropping it would hide that.
func TestSessionMenuOffersANewWorktreeWhereTheRepoIsKnown(t *testing.T) {
	m, _ := worktreePromptOS(t)

	for _, tc := range []struct {
		session string
		dim     bool
	}{
		{"tuios-feat-one", false},
		{"main", true},
	} {
		_, items := m.sessionMenu(tc.session)
		var row *ContextMenuItem
		for i := range items {
			if items[i].Action == "session_new_worktree" {
				row = &items[i]
				break
			}
		}
		if row == nil {
			t.Fatalf("the menu for %q has no worktree row", tc.session)
		}
		if row.Dim != tc.dim {
			t.Errorf("the worktree row on %q is dimmed %v, want %v", tc.session, row.Dim, tc.dim)
		}
	}

	// Without a daemon there is no listing to read, so the row is dimmed and
	// never a panic.
	m.DaemonClient = nil
	_, items := m.sessionMenu("tuios-feat-one")
	for _, it := range items {
		if it.Action == "session_new_worktree" && !it.Dim {
			t.Error("the worktree row is live with no daemon to ask")
		}
	}
}

// TestTheRepoHeaderMenuOffersANewWorktree: a repository's group header is the
// row that names the repository, so its menu offers a worktree of it.
func TestTheRepoHeaderMenuOffersANewWorktree(t *testing.T) {
	m, _ := worktreePromptOS(t)

	title, items := m.repoHeaderMenu("tuios")
	if title != "tuios" {
		t.Errorf("the menu is headed %q, want the repository the row names", title)
	}
	found := false
	for _, it := range items {
		if it.Action == "repo_new_worktree" {
			if it.Dim {
				t.Error("the worktree row is dimmed while the listing names the repository")
			}
			found = true
		}
	}
	if !found {
		t.Error("the repository header's menu has no worktree row")
	}

	// A repository the cache no longer names has nowhere to branch from.
	_, items = m.repoHeaderMenu("ghost")
	for _, it := range items {
		if it.Action == "repo_new_worktree" && !it.Dim {
			t.Error("the worktree row is live for a repository with no known checkout")
		}
	}
}

// TestTheWorktreeDialogOpensOnTheSessionsRepository: a session row opens the
// dialog on the repository the session's worktree record names, and a plain
// session is refused with a word and no dialog.
func TestTheWorktreeDialogOpensOnTheSessionsRepository(t *testing.T) {
	m, _ := worktreePromptOS(t)

	m.BeginWorktreePromptForSession("tuios-feat-one")
	if m.worktreePrompt == nil {
		t.Fatal("the dialog did not open on a worktree session")
	}
	if m.worktreePrompt.RepoRoot != "/src/tuios" {
		t.Errorf("the dialog opened on %q, want the record's repo root", m.worktreePrompt.RepoRoot)
	}
	if m.worktreePrompt.Input != "" {
		t.Errorf("the dialog opened with %q typed, want an empty field", m.worktreePrompt.Input)
	}
	m.WorktreePromptCancel()
	if m.WorktreePromptOpen() {
		t.Fatal("esc did not close the dialog")
	}

	m.BeginWorktreePromptForSession("main")
	if m.WorktreePromptOpen() {
		t.Error("the dialog opened on a session with no worktree record")
	}
}

// TestTheWorktreeDialogOpensOnTheRepoHeader: a repository's header resolves
// the main checkout from any member session's record.
func TestTheWorktreeDialogOpensOnTheRepoHeader(t *testing.T) {
	m, _ := worktreePromptOS(t)

	m.BeginWorktreePromptForRepo("docs")
	if m.worktreePrompt == nil || m.worktreePrompt.RepoRoot != "/src/docs" {
		t.Fatalf("the dialog opened as %+v, want it on /src/docs", m.worktreePrompt)
	}
}

// TestTheWorktreeDialogDerivesWhatEnterWillMake: the folder and the session
// name are derived from the branch as it is typed, the same way the daemon
// derives them, and the drawing says so.
func TestTheWorktreeDialogDerivesWhatEnterWillMake(t *testing.T) {
	m, _ := worktreePromptOS(t)
	m.BeginWorktreePromptForSession("tuios-feat-one")

	m.WorktreePromptType("feat/retry")
	want := worktree.PathFor(worktree.DefaultDir(), "/src/tuios", "feat/retry")
	if got := m.worktreePromptPath(); got != want {
		t.Errorf("the derived folder is %q, want %q", got, want)
	}

	content, _, _ := m.renderWorktreePrompt()
	if !strings.Contains(content, "tuios-feat-retry") {
		t.Errorf("the dialog does not show the derived session name:\n%s", content)
	}
	if !strings.Contains(content, shortenHome(want)) && !strings.Contains(content, want) {
		t.Errorf("the dialog does not show the derived folder:\n%s", content)
	}
}

// TestWorktreeVerbParamsCarryTheTypedName: an empty session field sends none,
// which is how the verb says "derive it", and a typed one rides as name — the
// verb's own optional.
func TestWorktreeVerbParamsCarryTheTypedName(t *testing.T) {
	plain := worktreeVerbParams("/src/tuios", "feat/retry", "")
	if _, ok := plain["name"]; ok {
		t.Errorf("an empty session name sent %v, want the verb to derive", plain)
	}
	named := worktreeVerbParams("/src/tuios", "feat/retry", "my-name")
	if named["name"] != "my-name" || named["repo"] != "/src/tuios" || named["branch"] != "feat/retry" {
		t.Errorf("the create sent %v, want repo, branch and the typed name", named)
	}
}

// TestTheWorktreeDialogTypesIntoTheFieldTheKeyboardIsIn: tab moves between the
// branch and the session name, and every field edit lands in the active one.
func TestTheWorktreeDialogTypesIntoTheFieldTheKeyboardIsIn(t *testing.T) {
	m, _ := worktreePromptOS(t)
	m.BeginWorktreePromptForSession("tuios-feat-one")

	m.WorktreePromptType("feat/retry")
	m.WorktreePromptNextField()
	m.WorktreePromptType("my-name")
	if got := m.WorktreePromptBranch(); got != "feat/retry" {
		t.Errorf("the branch field holds %q after typing into the name field", got)
	}
	if got := m.WorktreePromptSessionName(); got != "my-name" {
		t.Errorf("the name field holds %q, want what was typed", got)
	}

	// Clearing and backspacing act on the active field only.
	m.WorktreePromptClearInput()
	if got := m.WorktreePromptSessionName(); got != "" {
		t.Errorf("clear emptied %q, want the name field alone", got)
	}
	m.WorktreePromptType("again")
	m.WorktreePromptBackspace()
	if got := m.WorktreePromptSessionName(); got != "agai" {
		t.Errorf("backspace left %q, want the name trimmed a rune", got)
	}
	if got := m.WorktreePromptBranch(); got != "feat/retry" {
		t.Errorf("the branch field changed to %q", got)
	}
	m.WorktreePromptNextField()
	m.WorktreePromptBackspace()
	if got := m.WorktreePromptBranch(); got != "feat/retr" {
		t.Errorf("backspace on the branch field left %q", got)
	}
}

// TestTheWorktreeDialogSubmitsTheTypedName: a create with a session name in it
// closes the dialog and hands the verb both fields.
func TestTheWorktreeDialogSubmitsTheTypedName(t *testing.T) {
	m, _ := worktreePromptOS(t)
	m.BeginWorktreePromptForSession("tuios-feat-one")
	m.WorktreePromptType("feat/retry")
	m.WorktreePromptNextField()
	m.WorktreePromptType("my-name")

	if m.WorktreePromptSubmit() == nil {
		t.Fatal("the submit produced no create")
	}
	if m.WorktreePromptOpen() {
		t.Error("the dialog stayed up after handing the create off")
	}
}

// TestTheWorktreeDialogRefusesBeforeTheDaemon: an empty field and a branch git
// would refuse are answered in the dialog, which stays up with the refusal
// under the field. A branch the daemon can accept closes the dialog and hands
// the create to a command.
func TestTheWorktreeDialogRefusesBeforeTheDaemon(t *testing.T) {
	m, _ := worktreePromptOS(t)
	m.BeginWorktreePromptForSession("tuios-feat-one")

	if cmd := m.WorktreePromptSubmit(); cmd != nil {
		t.Error("an empty branch produced a create")
	}
	if !m.WorktreePromptOpen() || m.worktreePrompt.Err == "" {
		t.Fatalf("an empty branch did not say why it stayed: %+v", m.worktreePrompt)
	}

	m.WorktreePromptClearInput()
	m.WorktreePromptType("feat/../escape")
	if cmd := m.WorktreePromptSubmit(); cmd != nil {
		t.Error("a branch git would refuse produced a create")
	}
	if !m.WorktreePromptOpen() || m.worktreePrompt.Err == "" {
		t.Fatalf("an invalid branch did not say why it stayed: %+v", m.worktreePrompt)
	}

	m.WorktreePromptClearInput()
	m.WorktreePromptType("feat/retry")
	cmd := m.WorktreePromptSubmit()
	if cmd == nil {
		t.Fatal("a valid branch produced no create")
	}
	if m.WorktreePromptOpen() {
		t.Error("the dialog stayed up after handing the create off")
	}
}

// TestRepoRowRightClickOpensTheRepoMenu: a repository header's right-click
// builds the repository menu and carries the repository's name, the same carry
// its left-click reads to fold and unfold the group. The carry reaching
// session-lifecycle actions is not a hazard here, because the menu the header
// builds holds none of them; see TestRailWorktreeGroupHasNoSessionMenu.
func TestRepoRowRightClickOpensTheRepoMenu(t *testing.T) {
	m, tree := worktreePromptOS(t)
	lines := railPlain(t, m, tree)

	_, hit := railRowLine(t, m, lines, sidebarRowRepo, "tuios")
	m.openSidebarContextMenu(hit, hit.X0, hit.Y0)
	if m.ContextMenu == nil || m.ContextMenu.Target != CtxTargetRepo {
		t.Fatalf("the right-click opened %+v, want a repository menu", m.ContextMenu)
	}
	// The carry is the carry the dispatcher hands the action: the repository's
	// name, which the dialog opens on.
	if got := m.ContextMenuSelectedAction(); got != "repo_new_worktree" {
		t.Fatalf("the menu's first row is %q, want repo_new_worktree", got)
	}
	if m.menuSession != "tuios" {
		t.Errorf("the menu carries %q, want the repository the row named", m.menuSession)
	}
}

// TestARepoHeaderWithoutADaemonKeepsItsOldMenu: with no daemon, or no cached
// checkout of the repository, the header's right-click falls back to the rail's
// own settings, which is what it opened before the menu existed.
func TestARepoHeaderWithoutADaemonKeepsItsOldMenu(t *testing.T) {
	m, tree := worktreePromptOS(t)
	lines := railPlain(t, m, tree)
	_, hit := railRowLine(t, m, lines, sidebarRowRepo, "tuios")

	m.DaemonClient = nil
	m.openSidebarContextMenu(hit, hit.X0, hit.Y0)
	if m.ContextMenu == nil || m.ContextMenu.Target == CtxTargetRepo {
		t.Fatalf("a header with no daemon opened %+v, want the rail settings", m.ContextMenu)
	}
	m.CloseContextMenu()

	m.BeginWorktreePromptForRepo("tuios")
	if m.WorktreePromptOpen() {
		t.Error("the dialog opened with no daemon to read the listing from")
	}
}
