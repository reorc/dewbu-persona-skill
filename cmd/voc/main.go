package main

import "github.com/reorc/dewbu-persona-skill/internal/cli"

var (
	Version   = "dev"
	BuildTime = ""
	GitCommit = ""
)

func main() {
	cli.SetVersionInfo(Version, BuildTime, GitCommit)
	cli.Execute()
}
