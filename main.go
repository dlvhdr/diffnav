package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone"
	"github.com/muesli/termenv"

	"github.com/dlvhdr/diffnav/pkg/config"
	"github.com/dlvhdr/diffnav/pkg/ui"
)

func main() {
	zone.NewGlobal()

	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		fmt.Println("No diff, exiting")
		os.Exit(0)
	}

	if os.Getenv("DEBUG") == "true" {
		var fileErr error
		logFile, fileErr := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if fileErr != nil {
			fmt.Println("Error opening debug.log:", fileErr)
			os.Exit(1)
		}
		defer logFile.Close()

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
			log.Debug("🚀 Starting diffnav", "logFile", wd+string(os.PathSeparator)+logFile.Name())
		}
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
	p := tea.NewProgram(ui.New(input, cfg), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
