package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Bowery/slack"
)

var (
	channel    string
	token      string
	configPath string
	slackToken string
	tee        bool
	err        error
)

const usage = `Standard Slack

Usage: stdslack [options]

stdslack reads from standard input and posts the given
input as a message on slack.

Options:
  --channel, -c  Channel to post to.
  --token, -t    Slack auth token.
`

func init() {
	flag.StringVar(&channel, "channel", "", "")
	flag.StringVar(&channel, "c", "", "")
	flag.StringVar(&token, "token", "", "")
	flag.StringVar(&token, "t", "", "")
	flag.BoolVar(&tee, "tee", false, "")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, usage)
	}

	homeVar := "HOME"
	if runtime.GOOS == "windows" {
		homeVar = "USERPROFILE"
	}
	configPath = filepath.Join(os.Getenv(homeVar), ".stdslackconf")
}

func main() {
	flag.Parse()
	if token != "" {
		err = ioutil.WriteFile(configPath, []byte(token), 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Printf("Wrote token to %s\n", configPath)
		return
	}

	if channel == "" {
		fmt.Fprintln(os.Stderr, "A channel is required")
		os.Exit(1)
	}
	if channel[0] != '#' {
		channel = "#" + channel
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = errors.New("run `stdslack --token=YOUR_TOKEN` to set token before using")
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slackC := slack.NewClient(string(data))

	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Check if stdin is from a terminal (i.e. no input to read).
	if stat.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, "Content needs to be given to stdin to use")
		os.Exit(1)
	}

	if tee {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stdout, "%s\n", line)
			err = slackC.SendMessage(channel, line, "stdslack")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	} else {
		var content bytes.Buffer
		_, err = io.Copy(&content, os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		err = slackC.SendMessage(channel, content.String(), "stdslack")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
