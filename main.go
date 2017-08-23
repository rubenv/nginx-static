package main

import (
	"bufio"
	"html/template"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	// Config watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					err := process(tpl, cmd)
					if err != nil {
						log.Printf("Failed to process config: %s", err)
					}
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
				return
			}
		}
	}()

	err = watcher.Add(cfgPath)
	if err != nil {
		return err
	}

	return cmd.Wait()
}

type SiteConfig struct {
	Hostname string
	Path     string
}

func writeConfig(tpl *template.Template) error {
	cfg, err := os.Open(cfgPath)
	if err != nil {
		return err
	}
	defer cfg.Close()

	hosts := make([]SiteConfig, 0)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)

		if len(parts) == 2 {
			hosts = append(hosts, SiteConfig{
				Hostname: strings.TrimSpace(parts[0]),
				Path:     strings.TrimSpace(parts[1]),
			})
		}
	}

	fp, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer fp.Close()

	return tpl.Execute(fp, hosts)
}

func process(tpl *template.Template, cmd *exec.Cmd) error {
	err := writeConfig(tpl)
	if err != nil {
		return err
	}

	// Trigger reload of pgbouncer config
	return cmd.Process.Signal(syscall.SIGHUP)
}
