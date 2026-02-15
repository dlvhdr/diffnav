package cmd

import (
	"bufio"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/muesli/termenv"

	"github.com/dlvhdr/diffnav/pkg/config"
	"github.com/dlvhdr/diffnav/pkg/ui"
	"github.com/dlvhdr/diffnav/pkg/version"
)

//go:embed logo-diff-part.txt
var asciiArtDiffPart string

//go:embed logo-nav-part.txt
var asciiArtNavPart string

var logo = lipgloss.JoinHorizontal(lipgloss.Top,
	lipgloss.NewStyle().Foreground(lipgloss.Green).Render(asciiArtDiffPart),
	lipgloss.NewStyle().Foreground(lipgloss.Red).Render(asciiArtNavPart))

var rootCmd = &cobra.Command{
	Use:   "diffnav",
	Short: "DIFFNAV - a git diff pager based on delta but with a file tree, à la GitHub.",
	Long: "\n" + logo + lipgloss.NewStyle().Foreground(lipgloss.White).Render(
		"\na git diff pager based on delta\nbut with a file tree, à la GitHub"),
	Example: `# pipe into diffnav
git diff | diffnav

# use with the GitHub CLI
gh pr diff https://github.com/dlvhdr/gh-dash/pull/447 | diffnav

# set up as the global git diff pager
git config --global pager.diff diffnav
	`,
}

func Execute() {
	themeFunc := fang.WithColorSchemeFunc(func(
		ld lipgloss.LightDarkFunc,
	) fang.ColorScheme {
		def := fang.DefaultColorScheme(ld)
		def.DimmedArgument = ld(lipgloss.Black, lipgloss.White)
		def.Codeblock = ld(lipgloss.Color("#F1EFEF"), lipgloss.Color("#141417"))
		def.Title = lipgloss.Red
		def.Command = lipgloss.Green
		def.Program = lipgloss.Green
		return def
	})

	if err := fang.Execute(
		context.Background(),
		rootCmd,
		themeFunc,
		fang.WithVersion(version.Version),
		fang.WithNotifySignal(os.Interrupt)); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolP("side-by-side", "s", false, "Force side-by-side diff view")

	rootCmd.Flags().BoolP("unified", "u", false, "Force unified diff view")

	rootCmd.SetVersionTemplate("\n" + logo + "\n" + `{{printf "version %s\n" .Version}}`)

	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		// Parse CLI flags
		sideBySideFlag, err := cmd.Flags().GetBool("side-by-side")
		if err != nil {
			log.Fatal("Cannot parse the side-by-side flag", err)
		}
		unifiedFlag, err := cmd.Flags().GetBool("unified")
		if err != nil {
			log.Fatal("Cannot parse the unified flag", err)
		}

		helpFlag, err := cmd.Flags().GetBool("help")
		if err != nil {
			log.Fatal("Cannot parse the help flag", err)
		}

		zone.NewGlobal()

		stat, err := os.Stdin.Stat()
		if err != nil {
			panic(err)
		}

		if !helpFlag && stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
			fmt.Println("No diff, exiting")
			os.Exit(0)
		}

		if os.Getenv("DEBUG") == "true" {
			var fileErr error
			logFile, fileErr := os.OpenFile("debug.log",
				os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
			if fileErr != nil {
				fmt.Println("Error opening debug.log:", fileErr)
				os.Exit(1)
			}
			defer func() {
				if err := logFile.Close(); err != nil {
					log.Fatal("failed closing log file", "err", err)
				}
			}()

			if fileErr == nil {
				log.SetOutput(logFile)
				log.SetTimeFormat(time.Kitchen)
				log.SetReportCaller(true)
				log.SetLevel(log.DebugLevel)

				log.SetOutput(logFile)
				log.SetColorProfile(termenv.TrueColor)
				wd, err := os.Getwd()
				if err != nil {
					fmt.Println("Error getting current working dir", err)
					os.Exit(1)
				}
				log.Debug("🚀 Starting diffnav", "logFile",
					wd+string(os.PathSeparator)+logFile.Name())
			}
		} else {
			log.SetOutput(os.Stderr)
			log.SetLevel(log.FatalLevel)
		}

		reader := bufio.NewReader(os.Stdin)
		var b strings.Builder

		for {
			r, _, err := reader.ReadRune()
			if err != nil && err == io.EOF {
				break
			}
			_, err = b.WriteRune(r)
			if err != nil {
				fmt.Println("Error getting input:", err)
				os.Exit(1)
			}
		}

		input := ansi.Strip(b.String())
		if strings.TrimSpace(input) == "" {
			fmt.Println("No input provided, exiting")
			os.Exit(0)
		}
		cfg := config.Load()

		// Override config with CLI flags if specified
		if unifiedFlag {
			cfg.UI.SideBySide = false
		} else if sideBySideFlag {
			cfg.UI.SideBySide = true
		}

		ttyIn, _, err := tea.OpenTTY()
		if err != nil {
			log.Fatal(err)
		}
		p := tea.NewProgram(ui.New(input, cfg), tea.WithInput(ttyIn))

		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	}
}
