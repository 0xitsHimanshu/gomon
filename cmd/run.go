package cmd

import (
  "fmt"
  "log"
  "os"
  "os/exec"
  "path/filepath"
  "sync"
  
  "github.com/fatih/color"
  "github.com/fsnotify/fsnotify"
  "github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
  Use: "run [file]",
  Short: "Run and watch your Go app",
  Args: cobra.ExactArgs(1),
  Run: func(cmd *cobra.Command, args []string){
    targetFile := args[0]
    runAndWatch(targetFile)
  },
}

func init() {
  rootCmd.AddCommand(runCmd)
}

func runAndWatch(target string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	var mu sync.Mutex
	var cmdProc *exec.Cmd

	start := func() {
		color.Green("🚀 Starting %s\n", target)
		cmdProc = exec.Command("go", "run", target)
		cmdProc.Stdout = os.Stdout
		cmdProc.Stderr = os.Stderr
		if err := cmdProc.Start(); err != nil {
			color.Red("Error starting process: %v", err)
		}
	}

	restart := func() {
		mu.Lock()
		defer mu.Unlock()
		if cmdProc != nil && cmdProc.Process != nil {
			_ = cmdProc.Process.Kill()
		}
		color.Yellow("🔁 Restarting...")
		start()
	}

	// Start process initially
	start()

	// Watch all Go files recursively
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && (info.Name() == "vendor" || info.Name() == ".git") {
			return filepath.SkipDir
		}
		if info.IsDir() {
			watcher.Add(path)
		}
		return nil
	})

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if filepath.Ext(event.Name) == ".go" {
				fmt.Printf("📂 Change detected: %s\n", event)
				restart()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			color.Red("Watcher error: %v", err)
		}
	}
}
