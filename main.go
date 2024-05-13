package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"
)

func die(verbose bool, msg string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, msg+"\n", args...)
	} else {
		fmt.Fprint(os.Stderr, "error (use -v) ")
	}
	os.Exit(1)
}

const (
	timeLayout  = "2006-01-02T15:04:05Z07:00"
	breakString = "break"
	offString   = "off"
)

var (
	pStart    = flag.Bool("s", false, "start a timer")
	pBreak    = flag.Bool("b", false, "start a break")
	pOff      = flag.Bool("o", false, "turn off tracking")
	notify    = flag.Bool("n", false, "whether to run the notify command")
	notifyCmd = flag.String("nc", "tmux::display::-d::2000::'fin!'", "command to run for notifying (as a string separated by ::)")
	verbose   = flag.Bool("v", false, "show verbose errors")
)

func main() {
	currentUser, err := user.Current()

	if err != nil {
		die(true, "Error getting your home directory, explicitly specify the path for the local file pomm needs using -f")
	}

	var pomFileDefaultLoc string
	if err == nil {
		pomFileDefaultLoc = fmt.Sprintf("%s/.pomm", currentUser.HomeDir)
	}

	pomFile := flag.String("f", pomFileDefaultLoc, "location of the file pomm will use to store pomodoro status")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "pomm is a pomodoro progress indicator intended for tmux and similar terminal multiplexers.\n\nFlags:\n")
		flag.CommandLine.SetOutput(os.Stderr)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *pomFile == "" {
		die(*verbose, "-f cannot be empty")
	}

	if *pStart && *pBreak {
		die(*verbose, "-s and -b cannot both be true at the same time")
	}

	if *notifyCmd == "" {
		die(*verbose, "notify command cannot be empty")
	}

	var notifyCmdParts []string
	if *notify {
		notifyCmdParts = strings.Split(*notifyCmd, "::")
	}

	now := time.Now().UTC()

	_, err = os.Stat(*pomFile)

	// if starting/breaking/turning off, just write the timestamp/msg and exit
	if *pStart || *pBreak || *pOff {
		file, err := os.OpenFile(*pomFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			die(*verbose, "Error opening file at %q: %s", *pomFile, err)
		}
		defer file.Close()

		var wErr error

		if *pStart {
			timestamp := now.Format(timeLayout)
			_, wErr = file.WriteString(fmt.Sprintf("%s", timestamp))
		} else if *pBreak {

			_, wErr = file.WriteString(breakString)
		} else if *pOff {
			_, wErr = file.WriteString(offString)
		}
		if wErr != nil {
			die(*verbose, "Error writing to file %v: %q", *pomFile, wErr)
		}
		return
	}

	// report pomodoro progress
	fileContentsB, err := os.ReadFile(*pomFile)
	if err != nil {
		die(*verbose, "Error reading file at %v: %q", *pomFile, err)
	}

	fileContents := strings.TrimSpace(string(fileContentsB))

	if fileContents == offString {
		return
	} else if fileContents == breakString {
		fmt.Printf("break!\n")
		return
	}

	t, err := time.Parse(timeLayout, fileContents)
	if err != nil {
		die(*verbose, "Error parsing time:", err)
		return
	}

	diff := int(now.Sub(t).Seconds())
	chunks := diff / 150

	if chunks >= 10 {
		fmt.Print("\\o/")
		if *notify {
			_ = exec.Command(notifyCmdParts[0], notifyCmdParts[1:]...).Run()
		}
	} else {
		for i := 0; i < chunks; i++ {
			fmt.Printf("▪")
		}
		for i := 0; i < 10-chunks; i++ {
			fmt.Printf("▫")
		}
	}
}
