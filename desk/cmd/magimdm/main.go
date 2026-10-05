package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/5mil/magimdm-windows/internal/desk"
)

func main() {
	port := flag.Int("port", 8788, "local desk port")
	self := flag.Bool("self-test", false, "run the desk self-test and exit")
	noWindow := flag.Bool("no-window", false, "serve only")
	flag.Parse()
	if *self {
		os.Exit(desk.SelfTest())
	}
	d := desk.New(desk.DataDir())
	addr, err := d.Listen(*port)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("magimdm.exe listening on http://%s\n", addr)
	if *noWindow {
		select {}
	}
	url := "http://" + addr + "/"
	cmd := openWindow(url)
	if cmd == nil {
		fmt.Println("open", url)
		select {}
	}
	_ = cmd.Wait()
	time.Sleep(200 * time.Millisecond)
}

func openWindow(url string) *exec.Cmd {
	if runtime.GOOS != "windows" {
		cmd := exec.Command("xdg-open", url)
		_ = cmd.Start()
		return nil
	}
	edge := filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe")
	if _, err := os.Stat(edge); err != nil {
		edge = filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe")
	}
	if _, err := os.Stat(edge); err == nil {
		return exec.Command(edge, "--app="+url, "--window-size=980,680")
	}
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	_ = cmd.Start()
	return nil
}
