package filet

func usageExamples() []usageRow {
	return []usageRow{
		{cmd: "filet check .", detail: "", desc: ""},
		{cmd: "filet roast ./apps ./pkg", detail: "", desc: ""},
		{cmd: "filet clean -dry-run .", detail: "", desc: ""},
		{cmd: "filet docker .", detail: "", desc: ""},
		{cmd: "filet test ./apps/api", detail: "", desc: ""},
		{cmd: "filet fuzz -mode smoke -fuzztime 10s .", detail: "", desc: ""},
		{cmd: "filet gate -keep-going .", detail: "", desc: ""},
		{cmd: "filet init -preset relaxed", detail: "", desc: ""},
	}
}
