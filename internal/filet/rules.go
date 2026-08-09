package filet

// Rule documents a single check so `filet rules` can list what exists.
type Rule struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

var registry = []Rule{
	{"gen.file.long", "file exceeds limits.fileLines"},
	{"gen.file.funcs", "file declares more than limits.funcsPerFile functions"},
	{"gen.line.long", "line exceeds limits.lineLength"},
	{"gen.nesting", "block nesting exceeds limits.nesting"},
	{"gen.todo", "TODO/FIXME/XXX/HACK leading a line comment or annotated with : or ( (style.banTODO)"},
	{"gen.commented.code", "commented-out code"},
	{"gen.comment.inline", "comment trailing a line of code (style.banInlineComments)"},
	{"go.comment.inbody", "comment inside a function body (style.banInlineComments)"},
	{"gen.trailing.space", "trailing whitespace (style.banTrailingSpace)"},
	{"go.parse", "file does not parse"},
	{"go.file.funcs", "file declares more than limits.funcsPerFile functions (tests exempt)"},
	{"go.func.long", "function body exceeds limits.funcLines (blank and comment lines excluded)"},
	{"go.func.statements", "function exceeds limits.funcStatements"},
	{"go.func.params", "function exceeds limits.params"},
	{"go.func.returns", "function exceeds limits.returns"},
	{"go.func.complexity", "cognitive complexity exceeds limits.complexity"},
	{"go.return.naked", "naked return in a function with named results"},
	{"go.err.discarded", "every return value assigned to _"},
	{"go.panic", "panic() outside test code"},
	{"go.struct.fields", "struct exceeds limits.structFields"},
	{"go.interface.big", "interface exceeds limits.interfaceMethods"},
	{"go.doc.missing", "exported declaration without a doc comment"},
	{"go.global.mutable", "package-level mutable var (style.banGlobalMutable)"},
	{"go.init", "init() function (style.banInit)"},
	{"go.receiver.inconsistent", "same type uses different receiver names"},
	{"arch.dir.missing", "architecture.requiredDirs entry is absent"},
	{"arch.dir.forbidden", "path matches architecture.forbiddenDirs"},
	{"arch.depth", "path deeper than architecture.maxDepth"},
	{"arch.filename", "filename does not match architecture.fileNamePattern"},
	{"arch.import.forbidden", "import banned for this path by architecture.forbiddenImports"},
	{"docker.from.latest", "base image pinned to :latest"},
	{"docker.from.untagged", "base image has no tag"},
	{"docker.user.root", "container ends up running as root"},
	{"docker.add.local", "ADD used where COPY would do"},
	{"docker.apt.recommends", "apt-get install without --no-install-recommends"},
	{"docker.apt.cleanup", "apt lists not removed in the same layer"},
	{"docker.apt.upgrade", "apt-get upgrade makes the build unreproducible"},
	{"docker.pip.cache", "pip install without --no-cache-dir"},
	{"docker.npm.install", "npm install instead of npm ci"},
	{"docker.curl.pipe", "download piped straight into a shell"},
	{"docker.sudo", "sudo used inside the image"},
	{"docker.secret", "secret-looking value in ENV or ARG"},
	{"docker.exec.form", "CMD or ENTRYPOINT in shell form"},
	{"docker.layers", "more than 8 RUN instructions"},
	{"docker.copy.cachebust", "COPY . . before the dependency install step"},
	{"docker.dockerignore.missing", "no .dockerignore next to the Dockerfile"},
	{"docker.healthcheck.missing", "no HEALTHCHECK instruction"},
	{"docker.workdir.missing", "no WORKDIR instruction"},
	{"docker.multistage.missing", "toolchain base image with a single build stage"},
	{"docker.maintainer", "MAINTAINER is deprecated"},
}

// Rules returns every rule filet can emit, in a stable order.
func Rules() []Rule { return registry }
