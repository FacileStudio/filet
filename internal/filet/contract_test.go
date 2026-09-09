package filet

import (
	"slices"
	"testing"
)

// frozenRules is the published rule set. Other tools key off these strings: a
// disabled: list names them, and the line format prints them in brackets. A
// rename or a removal is a breaking change, so it has to be made here first,
// deliberately, rather than fall out of an edit somewhere else.
var frozenRules = []string{
	"arch.depth",
	"arch.dir.forbidden",
	"arch.dir.missing",
	"arch.file.missing",
	"arch.filename",
	"arch.import.forbidden",
	"docker.add.local",
	"docker.apt.cleanup",
	"docker.apt.recommends",
	"docker.apt.upgrade",
	"docker.copy.cachebust",
	"docker.curl.pipe",
	"docker.dockerignore.missing",
	"docker.exec.form",
	"docker.from.latest",
	"docker.from.untagged",
	"docker.healthcheck.missing",
	"docker.layers",
	"docker.maintainer",
	"docker.multistage.missing",
	"docker.npm.install",
	"docker.pip.cache",
	"docker.secret",
	"docker.sudo",
	"docker.user.root",
	"docker.workdir.missing",
	"gen.comment.inline",
	"gen.commented.code",
	"gen.file.funcs",
	"gen.file.long",
	"gen.line.long",
	"gen.nesting",
	"gen.todo",
	"gen.trailing.space",
	"go.comment.inbody",
	"go.doc.form",
	"go.doc.missing",
	"go.err.discarded",
	"go.file.funcs",
	"go.func.complexity",
	"go.func.long",
	"go.func.params",
	"go.func.returns",
	"go.func.statements",
	"go.global.mutable",
	"go.init",
	"go.interface.big",
	"go.panic",
	"go.parse",
	"go.receiver.inconsistent",
	"go.return.naked",
	"go.struct.fields",
	"go.leak.resource",
	"go.err.nilerr",
	"ts.func.complexity",
	"ts.func.long",
	"ts.func.params",
	"ts.func.statements",
	"ts.nesting",
}

func TestRuleIDsAreFrozen(t *testing.T) {
	var live []string
	for _, r := range Rules() {
		live = append(live, r.ID)
	}
	slices.Sort(live)

	for _, id := range frozenRules {
		if !slices.Contains(live, id) {
			t.Errorf("rule %q disappeared; removing a published id breaks every config that names it", id)
		}
	}
	for _, id := range live {
		if !slices.Contains(frozenRules, id) {
			t.Errorf("rule %q is new; add it to frozenRules to acknowledge it is now published", id)
		}
	}
}

func TestEveryRuleIsDocumentedAndUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range Rules() {
		if seen[r.ID] {
			t.Errorf("rule %q is registered twice", r.ID)
		}
		seen[r.ID] = true
		if r.Description == "" {
			t.Errorf("rule %q has no description, so `filet rules` cannot explain it", r.ID)
		}
	}
}

func TestEveryRuleHasItsOwnPunchline(t *testing.T) {
	for _, r := range Rules() {
		if len(punchlines[r.ID]) == 0 {
			t.Errorf("rule %q falls back to a generic roast; write it one", r.ID)
		}
	}
}
