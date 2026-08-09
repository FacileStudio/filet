package filet

import "strings"

func (s *dockerScan) instruction(in Instruction, add addLine) {
	low := strings.ToLower(in.Args)
	switch in.Cmd {
	case "FROM":
		s.froms++
		s.from(in, add)
	case "RUN":
		s.runs++
		checkRun(in.Line, low, add)
		if depInstallRe.MatchString(low) {
			s.installedAfter = in.Line
		}
	case "ADD":
		checkAdd(in.Line, low, add)
	case "COPY":
		s.copyAll(in)
	case "USER":
		s.user(in, low)
	case "WORKDIR":
		s.hasWork = true
	case "HEALTHCHECK":
		s.hasHealth = true
	case "MAINTAINER":
		add("docker.maintainer", in.Line, Info, "MAINTAINER is deprecated; use LABEL org.opencontainers.image.authors")
	case "ENV", "ARG":
		checkSecret(in, add)
	case "CMD", "ENTRYPOINT":
		checkExecForm(in, add)
	}
}

func checkAdd(line int, low string, add addLine) {
	if strings.Contains(low, "http") || strings.Contains(low, ".tar") || strings.Contains(low, ".tgz") {
		return
	}
	add("docker.add.local", line, Warn, "ADD for a local path; COPY does less and surprises less")
}

func (s *dockerScan) copyAll(in Instruction) {
	f := strings.Fields(in.Args)
	if len(f) >= 2 && f[0] == "." && s.copyAllLine == -1 {
		s.copyAllLine = in.Line
	}
}

func (s *dockerScan) user(in Instruction, low string) {
	s.hasUser = true
	s.lastUserRoot = 0
	if t := strings.TrimSpace(low); t == "root" || t == "0" {
		s.lastUserRoot = in.Line
	}
}

func checkSecret(in Instruction, add addLine) {
	v := strings.SplitN(in.Args, "=", 2)
	if len(v) != 2 || !secretKeyRe.MatchString(v[0]) || strings.TrimSpace(v[1]) == "" {
		return
	}
	add("docker.secret", in.Line, Error, "secret-looking value baked into the image layers")
}

func checkExecForm(in Instruction, add addLine) {
	if strings.HasPrefix(in.Args, "[") {
		return
	}
	add("docker.exec.form", in.Line, Warn,
		in.Cmd+" in shell form: PID 1 becomes /bin/sh and signals go nowhere")
}

func (s *dockerScan) from(in Instruction, add addLine) {
	image := strings.Fields(in.Args)
	if len(image) == 0 {
		return
	}
	ref := image[0]
	checkTag(ref, in.Line, add)
	if s.froms == 1 && compilerBase.MatchString(strings.TrimPrefix(ref, "docker.io/library/")) {
		s.firstFromRef, s.firstFromLine = ref, in.Line
	}
}

func checkTag(ref string, line int, add addLine) {
	tag := ""
	if i := strings.LastIndex(ref, ":"); i > strings.LastIndex(ref, "/") {
		tag = ref[i+1:]
	}
	switch {
	case strings.Contains(ref, "@sha256:"):
	case tag == "":
		add("docker.from.untagged", line, Error, ref+" has no tag, so it means :latest and your build is a time bomb")
	case tag == "latest":
		add("docker.from.latest", line, Error, ref+" pins to :latest, which pins to nothing")
	}
}
