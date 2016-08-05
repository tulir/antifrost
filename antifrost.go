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
	"fmt"
	flag "maunium.net/go/mauflag"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

var pipeStdout = flag.Make().Key("o", "stdout").Usage("Pipe stdout from child").Default("true").Bool()
var pipeStderr = flag.Make().Key("e", "stderr").Usage("Pipe stderr from child").Default("true").Bool()
var pipeSize = flag.Make().Key("s", "pipe-size").Usage("The maximum number of bytes to pipe at once (stdout/err)").Default("1024").Int()
var pipeStdin = flag.Make().Key("i", "stdin").Usage("Pipe stin to child").Default("true").Bool()

var autorestart = flag.Make().Key("r", "restart").Usage("Restart automatically if the program crashes").Default("false").Bool()

var tickerTime = flag.Make().Key("t", "time").Usage("The ticker interval in seconds").Default("30").Int64()
var tickerLimit = flag.Make().Key("l", "limit").Usage("The number of silent ticks to allow before restarting").Default("1").Int()

var quit = make(chan bool)
var output = make(chan bool, 1)

func main() {
	flag.MakeHelpFlag()
	flag.SetHelpTitles("antifrost - A wrapper to restarts programs if they freeze", "antifrost [-o] [-e] [-s=NUM] [-i] [-r] [-t=INTERVAL] [-l=TICKS] -- <command> [<args>...]")
	flag.Parse()
	if flag.CheckHelpFlag() {
		return
	}

	if flag.NArg() == 0 {
		os.Stderr.WriteString("Error: You must specify the command to run\n\n")
		flag.PrintHelp()
		return
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		quit <- true
	}()

	handleOutput()

	for {
		start(flag.Arg(0), flag.Args()[1:]...)
	}
}

func start(command string, args ...string) {
	cmd := exec.Command(command, args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if *pipeStdin {
		cmd.Stdin = os.Stdin
	}

	go cmd.Run()

	ticker := time.NewTicker(time.Duration(*tickerTime) * time.Second)
	ticked := 0
	for {
		select {
		case <-ticker.C:
			if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
				if *autorestart {
					fmt.Println("[Antifrost] Process exited! Restarting...")
					return
				}
				fmt.Println("[Antifrost] Process exited! Exiting...")
				os.Exit(0)
			} else if ticked < *tickerLimit {
				ticked++
			} else {
				if cmd.Process != nil {
					fmt.Println("[Antifrost] Process doesn't seem to be responding! Restarting...")
					cmd.Process.Kill()
				}
				return
			}
		case <-output:
			ticked = 0
		case <-quit:
			fmt.Print("[Antifrost] Interrupted! ")
			ticker.Stop()
			if cmd.ProcessState != nil && !cmd.ProcessState.Exited() {
				fmt.Println("Process still running! Sending SIGTERM...")
				cmd.Process.Signal(syscall.SIGTERM)
				go func() {
					time.Sleep(10 * time.Second)
					fmt.Println("[Antifrost] Process didn't exit! Sending SIGKILL and exiting...")
					cmd.Process.Kill()
					os.Exit(1)
				}()
				cmd.Wait()
				fmt.Println("[Antifrost] Process exited. Exiting...")
			} else {
				fmt.Print("\n")
			}
			os.Exit(0)
		}
	}
}

func handleOutput() {
	stdout := os.Stdout
	ro, wo, _ := os.Pipe()
	os.Stdout = wo

	go func() {
		if *pipeStdout {
			for {
				var rd = make([]byte, *pipeSize)
				n, _ := ro.Read(rd)
				if n > 0 {
					stdout.Write(rd)
					output <- true
				}
			}
		} else {
			for {
				n, _ := ro.Read(make([]byte, *pipeSize))
				if n > 0 {
					output <- true
				}
			}
		}
	}()

	stderr := os.Stderr
	re, we, _ := os.Pipe()
	os.Stderr = we

	go func() {
		if *pipeStderr {
			for {
				var rd = make([]byte, *pipeSize)
				n, _ := re.Read(rd)
				if n > 0 {
					stderr.Write(rd)
					output <- true
				}
			}
		} else {
			for {
				n, _ := re.Read(make([]byte, *pipeSize))
				if n > 0 {
					output <- true
				}
			}
		}
	}()
}
