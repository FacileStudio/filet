package filet

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *dockerScan) final(path string, add addLine) {
	if s.froms == 1 && s.firstFromRef != "" {
		add("docker.multistage.missing", s.firstFromLine, Warn,
			"single-stage build on "+s.firstFromRef+" ships the whole toolchain to production")
	}
	s.rootUser(add)
	s.missing(add)
	if s.runs > 8 {
		add("docker.layers", 1, Warn, fmt.Sprintf("%d RUN instructions; chain the related ones", s.runs))
	}
	if s.copyAllLine > 0 && s.installedAfter > s.copyAllLine {
		add("docker.copy.cachebust", s.copyAllLine, Warn,
			"COPY . . before the install step busts the dependency cache on every source change")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), ".dockerignore")); os.IsNotExist(err) {
		add("docker.dockerignore.missing", 1, Warn, "no .dockerignore, so .git and node_modules are in your build context")
	}
}

func (s *dockerScan) rootUser(add addLine) {
	switch {
	case !s.hasUser:
		add("docker.user.root", 1, Error, "no USER instruction, so the container runs as root")
	case s.lastUserRoot > 0:
		add("docker.user.root", s.lastUserRoot, Error, "final USER is root")
	}
}

func (s *dockerScan) missing(add addLine) {
	if !s.hasWork {
		add("docker.workdir.missing", 1, Info, "no WORKDIR; relative paths land wherever the base image left off")
	}
	if !s.hasHealth {
		add("docker.healthcheck.missing", 1, Info, "no HEALTHCHECK, so the orchestrator only knows the process exists")
	}
}

func checkRun(line int, low string, add addLine) {
	checkApt(line, low, add)
	if strings.Contains(low, "pip install") && !strings.Contains(low, "--no-cache-dir") {
		add("docker.pip.cache", line, Warn, "pip install without --no-cache-dir")
	}
	if strings.Contains(low, "npm install") && !strings.Contains(low, "-g ") {
		add("docker.npm.install", line, Warn, "npm install ignores the lockfile; use npm ci")
	}
	if curlPipeShell.MatchString(low) {
		add("docker.curl.pipe", line, Error, "piping a download straight into a shell is an unaudited install")
	}
	if strings.Contains(low, "sudo ") {
		add("docker.sudo", line, Warn, "sudo inside a container that already runs as root")
	}
}

func checkApt(line int, low string, add addLine) {
	if strings.Contains(low, "apt-get upgrade") || strings.Contains(low, "dist-upgrade") {
		add("docker.apt.upgrade", line, Warn, "apt-get upgrade makes the image unreproducible; pin packages instead")
	}
	if !strings.Contains(low, "apt-get install") && !strings.Contains(low, "apt install") {
		return
	}
	if !strings.Contains(low, "--no-install-recommends") {
		add("docker.apt.recommends", line, Warn, "apt-get install without --no-install-recommends")
	}
	if !strings.Contains(low, "rm -rf /var/lib/apt/lists") {
		add("docker.apt.cleanup", line, Warn, "apt lists never cleaned in the same layer, so the cache ships too")
	}
}
