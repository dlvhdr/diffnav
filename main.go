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

	"github.com/dlvhdr/diffnav/pkg/ui"
)

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		fmt.Println("Try piping in some text.")
		os.Exit(1)
	}

	var fileErr error
	logFile, fileErr := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if fileErr == nil {
		log.SetOutput(logFile)
		log.SetTimeFormat(time.Kitchen)
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
		defer logFile.Close()
		log.SetOutput(logFile)
		log.Debug("Starting diffnav, logging to debug.log")
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

	if os.Getenv("DEBUG") == "true" {
		logger, _ := tea.LogToFile("debug.log", "debug")
		defer logger.Close()
	}

	input := ansi.Strip(b.String())
	if strings.TrimSpace(input) == "" {
		fmt.Println("No input provided, exiting")
		os.Exit(1)
	}
	p := tea.NewProgram(ui.New(input), tea.WithMouseAllMotion())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
