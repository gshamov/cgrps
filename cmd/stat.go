// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"github.com/containerd/cgroups"
	"github.com/nsf/termbox-go"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

// statCmd represents the stat command
var statCmd = &cobra.Command{
	Use:   "stat [CGROUP]",
	Short: "cgroup stat",
	Long:  `cgroup stat.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if terminal.IsTerminal(0) {
			if len(args) < 1 {
				return errors.New("requires [CGROUP] or STDIN")
			}
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var c string

		if terminal.IsTerminal(0) {
			c = args[0]
		} else {
			b, _ := ioutil.ReadAll(os.Stdin)
			c = strings.TrimRight(string(b), "\n")
		}

		err := termbox.Init()
		if err != nil {
			panic(err)
		}
		defer termbox.Close()

		sch := make(chan bool)
		kch := make(chan termbox.Key)
		tch := make(chan bool)

		go drawLoop(sch, c)
		go keyEventLoop(kch)
		go timerLoop(tch)

		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

		for {
			select {
			case key := <-kch:
				mu.Lock()
				switch key {
				case termbox.KeyEsc, termbox.KeyCtrlC:
					mu.Unlock()
					return
				}
				mu.Unlock()
				sch <- true
				break
			case <-tch:
				mu.Lock()
				mu.Unlock()
				sch <- true
				break
			default:
				break
			}
		}

	},
}

var mu sync.Mutex
var timeSpan int = 1000 // ms

func timerLoop(tch chan bool) {
	for {
		tch <- true
		time.Sleep(time.Duration(timeSpan) * time.Millisecond)
	}
}

func keyEventLoop(kch chan termbox.Key) {
	for {
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey:
			kch <- ev.Key
		default:
		}
	}
}

func drawLine(x, y int, str string) {
	color := termbox.ColorDefault
	backgroundColor := termbox.ColorDefault
	runes := []rune(str)

	for i := 0; i < len(runes); i += 1 {
		termbox.SetCell(x+i, y, runes[i], color, backgroundColor)
	}
}

func drawLoop(sch chan bool, c string) {
	h := hierarchy(c)

	control, err := cgroups.Load(h, cgroups.StaticPath(c))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		<-sch
		mu.Lock()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		stats, err := control.Stat(cgroups.IgnoreNotExist)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		drawLine(0, 0, "Press ESC to exit.")
		drawLine(2, 1, fmt.Sprintf("%s", c))
		drawLine(2, 2, fmt.Sprintf("%v", stats))
		termbox.Flush()
		mu.Unlock()
	}
}

func init() {
	rootCmd.AddCommand(statCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
