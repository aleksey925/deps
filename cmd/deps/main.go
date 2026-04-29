package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	flag "github.com/spf13/pflag"

	"github.com/aleksey925/deps/internal/cli"
	"github.com/aleksey925/deps/internal/python"
	"github.com/aleksey925/deps/internal/ui"
)

var version = "0.0.0"

func main() {
	build := cli.NewBuildInfo(version)

	pythonPath := flag.StringP("python", "p", "", "path to Python interpreter")
	manager := flag.StringP("manager", "m", "", "package manager: pip or uv (auto-detected if not set)")
	showVersion := flag.BoolP("version", "v", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(build.Display())
		os.Exit(0)
	}

	env, err := python.DetectEnvironment(*pythonPath, *manager)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	m := ui.NewModel(env, build)

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
