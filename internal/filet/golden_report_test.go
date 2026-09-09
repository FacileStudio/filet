package filet

func goldenReport() Report {
	return Report{
		Findings: []Finding{
			newFinding("go.doc.missing", "apps/api/main.go", 12, Warn, "exported Run has no doc comment"),
			newFinding("docker.user.root", "Dockerfile", 0, Error, "no USER instruction"),
			newFinding("go.doc.missing", "apps/api/run.go", 3, Warn, "exported Do has no doc comment"),
		},
		Files: 3,
		Lines: 400,
	}
}
