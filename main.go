package main

import (
	"container/list"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/yangchenxing/go-nginx-conf-parser"
	"github.com/yangchenxing/go-testable-exit"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type log struct {
	path     string
	interval time.Duration
	keep     time.Duration
}

var (
	specialLogs     = make(map[string]log)
	defaultInterval time.Duration
	defaultKeep     time.Duration
	app             *cli.App
)

func init() {
	app = cli.NewApp()
	app.Name = "Nginx Log Rotater"
	app.Usage = "rotate nginx log and reload nginx cycly, also clean expired split files"
	app.Flags = []cli.Flag{
		&cli.DurationFlag{
			Name:  "interval",
			Value: time.Hour,
			Usage: "default rotation interval",
		},
		&cli.DurationFlag{
			Name:  "keep",
			Value: time.Hour * 24,
			Usage: "default keep time of log split files",
		},
		&cli.StringFlag{
			Name:  "pidfile",
			Value: "",
			Usage: "path to nginx pid file",
		},
		&cli.StringFlag{
			Name:  "workdir",
			Value: "",
			Usage: "path to nginx working directory",
		},
		&cli.StringFlag{
			Name:  "conffile",
			Value: "conf/nginx.conf",
			Usage: "path to nginx configure file",
		},
		&cli.StringSliceFlag{
			Name:  "special",
			Value: nil,
			Usage: "special log config in format \"path:interval:keep\"",
		},
	}
	app.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "list log file and corresponding interval and keep time",
			Action: doList,
		},
	}
}

func main() {
	exit.Exit(run())
}

func run() (code int) {
	defer func() {
		err := recover()
		if i, ok := err.(int); ok {
			code = i
		}
	}()
	app.Run(os.Args)
	return
}

func doList(context *cli.Context) {
	defaultInterval = context.GlobalDuration("interval")
	defaultKeep = context.GlobalDuration("keep")
	workDir := context.GlobalString("workdir")
	if err := parseSpecials(context.GlobalStringSlice("special")); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		panic(1)
	}
	confFile := context.GlobalString("conffile")
	logs, err := listLogs(workDir, confFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		panic(1)
	}
	for _, l := range logs {
		fmt.Printf("path: %q, interval: %s, keep: %s\n", l.path, l.interval, l.keep)
	}
}

func parseSpecials(specials []string) error {
	for _, special := range specials {
		sp := strings.Split(special, ":")
		if len(sp) != 3 {
			return fmt.Errorf("invalid special log parameter:", special)
		}
		interval, err := time.ParseDuration(sp[1])
		if err != nil {
			return fmt.Errorf("invalid special log parameter:", special)
		}
		keep, err := time.ParseDuration(sp[2])
		if err != nil {
			return fmt.Errorf("invalid special log parameter:", special)
		}
		specialLogs[sp[0]] = log{
			path:     sp[0],
			interval: interval,
			keep:     keep,
		}
	}
	return nil
}

func listLogs(workDir, confFile string) ([]log, error) {
	logList := list.New()
	if err := listLogsInFile(workDir, confFile, logList); err != nil {
		return nil, err
	}
	pathDedup := make(map[string]bool)
	for path := logList.Front(); path != nil; path = path.Next() {
		pathDedup[path.Value.(string)] = true
	}
	logs := make([]log, len(pathDedup))
	i := 0
	for path := range pathDedup {
		if l, found := specialLogs[path]; found {
			logs[i] = l
		} else {
			logs[i] = log{
				path:     path,
				interval: defaultInterval,
				keep:     defaultKeep,
			}
		}
		i++
	}
	return logs, nil
}

func listLogsInFile(workDir, confFile string, logs *list.List) error {
	path := filepath.Join(workDir, confFile)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	block, err := ncparser.Parse(content)
	if err != nil {
		return err
	}
	listLogsInBlock(block, workDir, logs)
	includes := list.New()
	listIncludeInBlock(block, includes)
	for inc := includes.Front(); inc != nil; inc = inc.Next() {
		if err := listLogsInFile(workDir, inc.Value.(string), logs); err != nil {
			return err
		}
	}
	return nil
}

func listIncludeInBlock(block ncparser.NginxConfigureBlock, includes *list.List) {
	for _, cmd := range block {
		if len(cmd.Words) >= 2 && cmd.Words[0] == "include" {
			includes.PushBack(cmd.Words[1])
		}
		if len(cmd.Block) > 0 {
			listIncludeInBlock(cmd.Block, includes)
		}
	}
}

func listLogsInBlock(block ncparser.NginxConfigureBlock, workDir string, logs *list.List) {
	for _, cmd := range block {
		if len(cmd.Words) >= 2 && (cmd.Words[0] == "access_log" || cmd.Words[0] == "error_log") &&
			cmd.Words[1] != "off" {
			logs.PushBack(filepath.Join(workDir, cmd.Words[1]))
		}
		if len(cmd.Block) > 0 {
			listLogsInBlock(cmd.Block, workDir, logs)
		}
	}
}
