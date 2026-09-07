package filet

import "hash/fnv"

var punchlines = map[string][]string{
	"gen.file.long": {
		"This file has more lines than the codebase has tests.",
		"At this length it is not a module, it is a memoir.",
		"Scrolling through this file counts as cardio.",
	},
	"gen.file.funcs": {
		"This file is not a module, it is a neighbourhood.",
		"Five functions was the budget. This is a spending problem.",
		"Somewhere past function six, cohesion left the building.",
	},
	"go.file.funcs": {
		"One file, many jobs, zero boundaries.",
		"If it needs this many functions, it needs a second file.",
		"The package is the unit of reuse. Use more of it.",
	},
	"gen.line.long": {
		"This line needs a boarding pass to reach the right margin.",
		"Word wrap was invented for a reason and you found it.",
		"Horizontal scrolling: the feature nobody asked for.",
	},
	"gen.nesting": {
		"Another level and this qualifies as a dream within a dream.",
		"Indentation this deep needs its own pressure suit.",
		"Somewhere in there is a business rule. Send a search party.",
	},
	"gen.todo": {
		"A TODO is a promise nobody intends to keep.",
		"This marker has outlived three product managers.",
		"git blame is going to be very funny here.",
	},
	"gen.commented.code": {
		"Version control exists. Delete it and let go.",
		"Commented-out code is a shrine to a decision you never made.",
		"If you need it back, git remembers. It always remembers.",
	},
	"gen.comment.inline": {
		"If the line needs a comment, the line needs a rename.",
		"Comments rot. Names compile.",
	},
	"go.comment.inbody": {
		"A comment in the body is a function begging to be split.",
		"Explaining the code inline means the code did not explain itself.",
	},
	"gen.trailing.space": {
		"Invisible characters, visible consequences.",
		"Your diff is 40% whitespace and 60% regret.",
	},
	"go.parse": {
		"It does not compile. Everything else is optimism.",
	},
	"go.func.long": {
		"This function does not have a responsibility, it has a portfolio.",
		"By line 80 the function has forgotten its own name.",
		"Split this before it develops opinions.",
	},
	"go.func.params": {
		"That is not a signature, that is a customs declaration.",
		"Pass a struct. Your future self will send a thank-you note.",
		"Nobody has ever called this correctly on the first try.",
	},
	"go.func.returns": {
		"Returning this many values is just a struct with commitment issues.",
		"The caller now needs a spreadsheet to unpack this.",
	},
	"go.func.complexity": {
		"The branches in here form a small but hostile ecosystem.",
		"Nested this deep, the control flow needs a table of contents.",
		"Every new condition here is a bug applying for a visa.",
	},
	"go.func.statements": {
		"That is not a function, that is an itinerary.",
		"Count the statements, then count how many belong together. Different numbers.",
	},
	"go.return.naked": {
		"Naked returns: mystery meat, but for control flow.",
		"Somewhere above, a named variable quietly changed. Good luck.",
	},
	"go.err.discarded": {
		"Discarding every return value is optimism with a compiler flag.",
		"The error you ignored is the incident you will page for.",
		"_ = doThing() is how outages introduce themselves.",
	},
	"go.panic": {
		"panic() is a very loud way to say 'not my problem'.",
		"Turning a caller's bad input into a process death is a bold API choice.",
	},
	"go.struct.fields": {
		"This struct is not a type, it is a filing cabinet.",
		"Somewhere in these fields, two of them mean the same thing.",
	},
	"go.interface.big": {
		"The bigger the interface, the weaker the abstraction. Rob Pike said it, not me.",
		"Nobody will ever implement this twice, which defeats the point.",
	},
	"go.doc.missing": {
		"Exported and undocumented: a gift with no instructions.",
		"Future readers will guess. They will guess wrong.",
	},
	"go.doc.form": {
		"godoc puts the name first so the index reads like a sentence. Yours does not.",
		"\"Returns the thing\" is not searchable. The identifier name is.",
	},
	"go.global.mutable": {
		"Package-level mutable state: the race condition's favourite hotel.",
		"Global state is just a data race with a longer fuse.",
	},
	"go.init": {
		"init() is code that runs before you can reason about it.",
		"Import side effects: spooky action at a distance, but in production.",
	},
	"go.receiver.inconsistent": {
		"Pick one receiver name. The type is the same type on every method.",
	},
	"arch.dir.missing": {
		"The architecture doc says this exists. The filesystem disagrees.",
	},
	"arch.dir.forbidden": {
		"This directory was explicitly banned and it came back anyway.",
		"A utils/ directory is where cohesion goes to die.",
	},
	"arch.file.missing": {
		"The convention is only a convention where somebody checks it.",
		"Seven directories follow this pattern. This one is improvising.",
	},
	"arch.depth": {
		"At this depth the import path is longer than the file.",
		"Directory archaeology should not be part of onboarding.",
	},
	"arch.filename": {
		"Naming conventions apply to everyone, including this file.",
	},
	"arch.import.forbidden": {
		"That import punches straight through the layer you drew on the whiteboard.",
		"The dependency arrow now points the wrong way. The architecture diagram is fiction.",
	},
	"docker.from.latest": {
		":latest is not a version, it is a mood.",
		"Reproducible builds, except for the part where the base image changes overnight.",
	},
	"docker.from.untagged": {
		"No tag means :latest, which means whatever Docker Hub felt like today.",
	},
	"docker.user.root": {
		"Running as root: one container escape from a very long weekend.",
		"The container is root, the host is nervous.",
	},
	"docker.add.local": {
		"ADD also unpacks archives and fetches URLs. You wanted none of that.",
	},
	"docker.apt.recommends": {
		"You just installed a package manager's idea of 'related interests'.",
		"200MB of recommended packages nobody recommended.",
	},
	"docker.apt.cleanup": {
		"The apt cache is now a permanent resident of your image.",
	},
	"docker.apt.upgrade": {
		"apt-get upgrade means the image built today and the image built tomorrow are strangers.",
	},
	"docker.pip.cache": {
		"pip's wheel cache is now shipping to production for free.",
	},
	"docker.npm.install": {
		"npm install in a Dockerfile: the lockfile was right there.",
	},
	"docker.curl.pipe": {
		"curl | sh is trusting a stranger with your root shell.",
	},
	"docker.sudo": {
		"sudo, inside a container, already running as root. Belt, braces, and a second belt.",
	},
	"docker.secret": {
		"That secret is in a layer forever. Layers do not forget, even after you delete the line.",
		"Anyone who can pull this image can read that value.",
	},
	"docker.exec.form": {
		"Shell form means PID 1 is /bin/sh and SIGTERM goes into the void.",
		"Your graceful shutdown is neither graceful nor a shutdown.",
	},
	"docker.layers": {
		"Every RUN is a layer, and every layer is a small monument to a decision.",
	},
	"docker.copy.cachebust": {
		"One character changed in a comment and you are reinstalling the internet.",
	},
	"docker.dockerignore.missing": {
		"Your build context includes .git, node_modules, and probably a .env.",
	},
	"docker.healthcheck.missing": {
		"'The process is running' is not the same as 'the service works'.",
	},
	"docker.workdir.missing": {
		"Relative paths without a WORKDIR is a treasure hunt with no map.",
	},
	"docker.multistage.missing": {
		"You shipped the compiler to production. It will not compile anything there.",
	},
	"docker.maintainer": {
		"MAINTAINER has been deprecated longer than some of your dependencies have existed.",
	},
	"go.leak.resource": {
		"An unclosed resource is a leak waiting to happen.",
		"Close() is not optional. The GC will not save you.",
		"Every open handle is a promise to the OS. Keep it.",
	},
	"go.err.nilerr": {
		"Handling an error then returning nil is how silent failures are born.",
		"The if-block is there for a reason. Use it or delete it.",
		"nil, nil compiles but does not compute.",
	},
}

var fallback = []string{
	"This is technically code.",
	"It works, which is the nicest thing available to say.",
	"Somebody had a deadline and it shows.",
}

// Roast fills in a punchline for each finding, chosen deterministically from its rule pool.
func Roast(findings []Finding) []Finding {
	for i, f := range findings {
		pool, ok := punchlines[f.Rule]
		if !ok || len(pool) == 0 {
			pool = fallback
		}
		h := fnv.New32a()
		h.Write([]byte(f.Rule))
		h.Write([]byte(f.File))
		h.Write([]byte{byte(f.Line), byte(f.Line >> 8)})
		findings[i].Roast = pool[int(h.Sum32())%len(pool)]
	}
	return findings
}
