package main

import (
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	yaml "gopkg.in/yaml.v1"

	"github.com/fsnotify/fsnotify"
)

const (
	cfgPath = "/etc/nginx-static/config"
	tplPath = "/etc/nginx/nginx.conf.tpl"
	outPath = "/etc/nginx/nginx.conf"
)

func main() {
	err := do()
	if err != nil {
		log.Fatal(err)
	}
}

func do() error {
	tpl, err := template.ParseFiles(tplPath)
	if err != nil {
		return err
	}

	// Write initial config
	err = writeConfig(tpl)
	if err != nil {
		return err
	}

	// Start nginx
	cmd := exec.Command("/usr/sbin/nginx", "-g", "daemon off;")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		return err
	}

	// Shutdown listener
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM)

	go func() {
		for {
			done, err := watchEvents(tpl, cmd, stop)
			if err != nil {
				log.Println(err)
			}
			if done {
				return
			}
		}
	}()

	return cmd.Wait()
}

func watchEvents(tpl *template.Template, cmd *exec.Cmd, stop chan os.Signal) (bool, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return false, err
	}
	defer watcher.Close()

	err = watcher.Add(cfgPath)
	if err != nil {
		return false, err
	}

	select {
	case <-watcher.Events:
		err := process(tpl, cmd)
		if err != nil {
			log.Printf("Failed to process config: %s", err)
		}
	case err := <-watcher.Errors:
		log.Printf("Config watch error: %s", err)
	case <-stop:
		// Wait for a while
		time.Sleep(30 * time.Second)
		err := cmd.Process.Signal(syscall.SIGINT)
		if err != nil {
			log.Println(err)
		}
		return true, nil
	}

	return false, nil
}

type SiteConfig struct {
	Host     string `yaml:"host"`
	Root     string `yaml:"root"`
	Redirect string `yaml:"redirect"`
}

func writeConfig(tpl *template.Template) error {
	cfg, err := os.Open(cfgPath)
	if err != nil {
		return err
	}
	defer cfg.Close()

	hosts, err := getConfig(cfg)
	if err != nil {
		return err
	}

	fp, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer fp.Close()

	return tpl.Execute(fp, hosts)
}

func getConfig(in io.Reader) ([]SiteConfig, error) {
	hosts := make([]SiteConfig, 0)

	b, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(b, &hosts)
	if err != nil {
		return nil, err
	}

	return hosts, nil
}

func process(tpl *template.Template, cmd *exec.Cmd) error {
	err := writeConfig(tpl)
	if err != nil {
		return err
	}

	// Trigger reload of config
	return cmd.Process.Signal(syscall.SIGHUP)
}
