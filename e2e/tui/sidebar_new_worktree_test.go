package tuie2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Gaurav-Gosain/tuios/internal/testutil"
	"github.com/Gaurav-Gosain/tuitest"
)

// TestRailRightClickMakesAWorktree drives the dialog the way a person meets
// it: a throwaway repository, a worktree session the CLI made so the rail has
// a repository row, a right-click on that row, the branch and the session name
// typed into the dialog, and a daemon whose ls is the authority on what came
// out of it. The client should end up attached to the session it named.
func TestRailRightClickMakesAWorktree(t *testing.T) {
	base := t.TempDir()
	killDaemon(t, base)
	repo := testutil.GitRepo(t)
	t.Setenv("TUIOS_WORKTREE_DIR", filepath.Join(base, "worktrees"))

	// A session that is not a worktree, which is what this client attaches to,
	// and one CLI-made worktree so the rail has the repository row to click.
	// The plain one starts the daemon, so it is created from a directory in no
	// repository.
	if out, err := tuiosCLIIn(t, base, t.TempDir(), "new", "plain", "--detach"); err != nil {
		t.Fatalf("create the plain session: %v: %s", err, out)
	}
	if out, err := tuiosCLI(t, base, "worktree", "new", "feat/one", "--repo", repo, "--detach"); err != nil {
		t.Fatalf("worktree new feat/one: %v: %s", err, out)
	}

	term := startIn(t, base, startOpts{args: []string{"attach", "plain"}})
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return countWindows(s) == 1
	}, bootTimeout); err != nil {
		t.Fatalf("client never attached: %v\n%s", err, term.Snapshot())
	}
	windowManagementMode(t, term)
	toggleSidebarViaPalette(t, term)
	waitForAll(t, term, uiTimeout, "sidebar with the repository row", sidebarHeader, "repo")

	// Right-click the repository's group header and take its worktree row.
	col, row := findOnScreen(t, term, "repo")
	mouseClick(t, term, col, row, tuitest.MouseRight, 0)
	if err := term.WaitForText("New worktree...", uiTimeout); err != nil {
		t.Fatalf("the repository row's menu offered no worktree: %v\n%s", err, term.Snapshot())
	}
	col, row = findOnScreen(t, term, "New worktree...")
	mouseClick(t, term, col, row, tuitest.MouseLeft, 0)
	if err := term.WaitForText("new worktree", uiTimeout); err != nil {
		t.Fatalf("the menu row did not open the dialog: %v\n%s", err, term.Snapshot())
	}

	// Type the branch. The session field's empty state reads as the name the
	// daemon will derive, which is the assertion that the derivation the rail
	// shows is the derivation the daemon does.
	if err := term.SendKeys("feat/ui"); err != nil {
		t.Fatalf("type the branch: %v", err)
	}
	if err := term.WaitForText("repo-feat-ui", uiTimeout); err != nil {
		t.Fatalf("the dialog did not derive the session name from the branch: %v\n%s", err, term.Snapshot())
	}
	saveFrame(t, term, "rail-worktree-dialog")

	// Tab to the session name and type one, so the create is by the name the
	// user chose and not the derived one.
	if err := term.SendKeys("\t"); err != nil {
		t.Fatalf("tab to the session name: %v", err)
	}
	if err := term.SendKeys("custom"); err != nil {
		t.Fatalf("type the session name: %v", err)
	}
	if err := term.SendKeys(tuitest.Enter); err != nil {
		t.Fatalf("submit the dialog: %v", err)
	}

	// The daemon is the authority on what was made: the session by the name
	// the user typed, the worktree on disk, and no session for the derived
	// name that stopped being the answer the moment a name was typed.
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return hasName(sessionNames(t, base), "custom")
	}, uiTimeout); err != nil {
		t.Fatalf("tuios never listed the created session: %v\n%s", err, term.Snapshot())
	}
	names := sessionNames(t, base)
	if !hasName(names, "repo-feat-one") {
		t.Errorf("tuios ls lists %v, want the CLI-made worktree session untouched", names)
	}
	if hasName(names, "repo-feat-ui") {
		t.Errorf("tuios ls lists %v, want the typed name in place of the derived one", names)
	}
	if out, err := exec.Command("git", "-C", filepath.Join(base, "worktrees", "repo", "feat-ui"), "branch", "--show-current").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "feat/ui" {
		t.Errorf("the worktree on disk is not on feat/ui: %v: %s", err, out)
	}

	// The new session joins the repository's group on the rail once the
	// client's listing carries it — the rail is the screen the user is looking
	// at, and it is the first place the create shows.
	if err := term.WaitFor(func(s tuitest.Screen) bool {
		return railLineWith(s, "feat/ui") >= 0
	}, uiTimeout); err != nil {
		t.Fatalf("the rail never showed the created worktree: %v\n%s", err, term.Snapshot())
	}

	alive(t, term, "after making a worktree from the rail")
}
