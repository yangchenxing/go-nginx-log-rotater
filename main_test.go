package main

import (
	"github.com/yangchenxing/go-testable-exit"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestParseSpecials(t *testing.T) {
	if err := parseSpecials([]string{"a:b:c:d"}); err == nil {
		t.Error("unexpected success")
		return
	}
	if err := parseSpecials([]string{"foo:m:1h"}); err == nil {
		t.Error("unexpected success")
		return
	}
	if err := parseSpecials([]string{"foo:1m:h"}); err == nil {
		t.Error("unexpected success")
		return
	}
	specialLogs = make(map[string]log)
	if err := parseSpecials([]string{"foo:1m:1h", "bar:1h:24h"}); err != nil {
		t.Error("unexpected fail:", err.Error())
		return
	}
	if len(specialLogs) != 2 {
		t.Error("unexpected result:", specialLogs)
		return
	}
	l := specialLogs["foo"]
	if l.path != "foo" || l.interval != time.Minute || l.keep != time.Hour {
		t.Error("unexpected result:", specialLogs)
		return
	}
	l = specialLogs["bar"]
	if l.path != "bar" || l.interval != time.Hour || l.keep != time.Hour*24 {
		t.Error("unexpected result:", specialLogs)
		return
	}
}

func TestListLogs(t *testing.T) {
	content1 := []byte(`
http {

include 2.conf;

server {
    listen 80;
    access_log logs/access1.log;
}
}`)
	content2 := []byte(`
server {
    listen 81;
    error_log logs/error2.log;
}`)
	specialLogs = map[string]log{
		"logs/error2.log": log{
			path:     "logs/error2.log",
			interval: time.Hour,
			keep:     time.Hour * 24,
		},
	}
	if err := ioutil.WriteFile("1.conf", content1, 0755); err != nil {
		t.Error("save 1.conf fail:", err.Error())
		return
	}
	defer os.Remove("1.conf")
	if err := ioutil.WriteFile("2.conf", content2, 0755); err != nil {
		t.Error("save 2.conf fail:", err.Error())
		return
	}
	defer os.Remove("2.conf")
	if logs, err := listLogs(".", "1.conf"); err != nil {
		t.Error("listLogs fail:", err.Error())
		return
	} else if len(logs) != 2 || !(logs[0].path == "logs/access1.log" && logs[1].path == "logs/error2.log" || logs[1].path == "logs/access1.log" && logs[0].path == "logs/error2.log") {
		t.Error("unexpected result:", logs)
		return
	}
	// fail case 1
	if logs, err := listLogs(".", "3.conf"); err == nil {
		t.Error("unexpected result:", logs)
		return
	}
	// fail case 2
	content3 := []byte(`
include 4.conf;
`)
	content4 := []byte(`
not a conf`)
	if err := ioutil.WriteFile("3.conf", content3, 0755); err != nil {
		t.Error("save 3.conf fail:", err.Error())
		return
	}
	defer os.Remove("3.conf")
	if err := ioutil.WriteFile("4.conf", content4, 0755); err != nil {
		t.Error("save 4.conf fail:", err.Error())
		return
	}
	defer os.Remove("4.conf")
	if logs, err := listLogs(".", "3.conf"); err == nil {
		t.Error("unexpected result:", logs)
		return
	}
}

func TestMainList(t *testing.T) {
	content1 := []byte(`
http {

include 2.conf;

server {
    listen 80;
    access_log logs/access1.log;
}
}`)
	content2 := []byte(`
server {
    listen 81;
    error_log logs/error2.log;
}`)
	specialLogs = map[string]log{
		"logs/error2.log": log{
			path:     "logs/error2.log",
			interval: time.Hour,
			keep:     time.Hour * 24,
		},
	}
	if err := ioutil.WriteFile("1.conf", content1, 0755); err != nil {
		t.Error("save 1.conf fail:", err.Error())
		return
	}
	defer os.Remove("1.conf")
	if err := ioutil.WriteFile("2.conf", content2, 0755); err != nil {
		t.Error("save 2.conf fail:", err.Error())
		return
	}
	defer os.Remove("2.conf")
	os.Args = []string{os.Args[0], "-conffile", "1.conf", "list"}
	code := run()
	if code != 0 {
		t.Error("unexpected exit code:", code)
		return
	}
	// test invalid special
	os.Args = []string{os.Args[0], "-conffile", "1.conf", "-special", "nothong", "list"}
	code = run()
	if code != 1 {
		t.Error("unexpected exit code:", code)
		return
	}
	// test invalid conffile
	os.Args = []string{os.Args[0], "-conffile", "3.conf", "list"}
	code = run()
	if code != 1 {
		t.Error("unexpected exit code:", code)
		return
	}
}

func TestMain(t *testing.T) {
	content1 := []byte(`
http {

include 2.conf;

server {
    listen 80;
    access_log logs/access1.log;
}
}`)
	content2 := []byte(`
server {
    listen 81;
    error_log logs/error2.log;
}`)
	specialLogs = map[string]log{
		"logs/error2.log": log{
			path:     "logs/error2.log",
			interval: time.Hour,
			keep:     time.Hour * 24,
		},
	}
	if err := ioutil.WriteFile("1.conf", content1, 0755); err != nil {
		t.Error("save 1.conf fail:", err.Error())
		return
	}
	defer os.Remove("1.conf")
	if err := ioutil.WriteFile("2.conf", content2, 0755); err != nil {
		t.Error("save 2.conf fail:", err.Error())
		return
	}
	defer os.Remove("2.conf")
	os.Args = []string{os.Args[0], "-conffile", "1.conf", "list"}
	exit.UsePanic = true
	defer func() {
		if err := recover(); err != nil {
			if e, ok := err.(exit.Error); ok && int(e) != 0 {
				t.Error("unexpected exit status:", int(e))
			}
		}
	}()
	main()
}
