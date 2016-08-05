// antifrost - A wrapper to restarts programs if they freeze
// Copyright (C) 2016 Tulir Asokan
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
	"bufio"
	flag "maunium.net/go/mauflag"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var pipeStdout = flag.Make().Key("o", "stdout").Usage("Pipe stdout from child").Default("true").Bool()
var pipeStderr = flag.Make().Key("e", "stderr").Usage("Pipe stderr from child").Default("true").Bool()
var pipeStdin = flag.Make().Key("i", "stdin").Usage("Pipe stin to child").Default("true").Bool()
var autorestart = flag.Make().Key("r", "restart").Usage("Restart automatically if the program crashes").Default("false").Bool()

var tickerTime = flag.Make().Key("t", "time").Usage("The ticker interval in seconds").Default("30").Int64()
var tickerLimit = flag.Make().Key("l", "limit").Usage("The number of silent ticks to allow before restarting").Default("1").Int()

var quit = make(chan bool)

func main() {
	flag.MakeHelpFlag()
	flag.SetHelpTitles("antifrost - A wrapper to restarts programs if they freeze", "antifrost [-o]")
	flag.Parse()
	flag.CheckHelpFlag()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		quit <- true
	}()

	for {
		start(flag.Arg(0), flag.Args()[1:]...)
	}
}

func start(command string, args ...string) {
	cmd := exec.Command(command, args...)

	if *pipeStdout {
		cmd.Stdout = os.Stdout
	}
	if *pipeStderr {
		cmd.Stderr = os.Stderr
	}
	if *pipeStdin {
		cmd.Stdin = os.Stdin
	}

	cmd.Start()

	stdout := make(chan bool, 1)
	go func() {
		r, _ := cmd.StdoutPipe()
		reader := bufio.NewReader(r)
		for {
			reader.ReadString('\n')
			stdout <- true
		}
	}()

	ticker := time.NewTicker(time.Duration(*tickerTime) * time.Second)
	ticked := 0
	for {
		select {
		case <-ticker.C:
			if cmd.ProcessState.Exited() {
				if *autorestart {
					return
				}
				os.Exit(0)
			} else if ticked < *tickerLimit {
				ticked++
			} else {
				cmd.Process.Kill()
				return
			}
		case <-stdout:
			ticked = 0
		case <-quit:
			ticker.Stop()
			if !cmd.ProcessState.Exited() {
				cmd.Process.Signal(syscall.SIGTERM)
				go func() {
					time.Sleep(10 * time.Second)
					cmd.Process.Kill()
					os.Exit(1)
				}()
				cmd.Wait()
			}
			os.Exit(0)
		}
	}
}
