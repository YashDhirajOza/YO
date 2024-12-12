package main

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	message string
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("\n%s\n\nPress 'q' to quit.\n", m.message)
}

func main() {
	// Initialize CLI tool
	if len(os.Args) < 2 {
		startUI("Usage: yo <command>")
		return
	}

	command := os.Args[1]

	switch command {
	case "init":
		if err := yoInit(); err != nil {
			startUI(fmt.Sprintf("Error initializing repository: %v", err))
			return
		}
		startUI("Initialized empty Yo repository successfully!")
	case "add":
		if len(os.Args) < 3 {
			startUI("Usage: yo add <file>")
			return
		}
		file := os.Args[2]
		if err := yoAdd(file); err != nil {
			startUI(fmt.Sprintf("Error adding file: %v", err))
			return
		}
		startUI(fmt.Sprintf("Added %s to staging area.", file))
	case "commit":
		if len(os.Args) < 3 {
			startUI("Usage: yo commit <message>")
			return
		}
		message := strings.Join(os.Args[2:], " ")
		if err := yoCommit(message); err != nil {
			startUI(fmt.Sprintf("Error committing changes: %v", err))
			return
		}
		startUI("Changes committed successfully!")
	case "log":
		log, err := yoLog()
		if err != nil {
			startUI(fmt.Sprintf("Error displaying log: %v", err))
			return
		}
		startUI(log)
	default:
		startUI(fmt.Sprintf("Unknown command: %s", command))
	}
}

func yoInit() error {
	// Create the .yo directory
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	repoPath := filepath.Join(currentDir, ".yo")
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		return fmt.Errorf(".yo directory already exists")
	}

	if err := os.MkdirAll(filepath.Join(repoPath, "objects"), 0755); err != nil {
		return fmt.Errorf("failed to create .yo directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(repoPath, "logs"), 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	return nil
}

func yoAdd(file string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	hash := hashObject(string(content))
	repoPath, _ := os.Getwd()
	objectPath := filepath.Join(repoPath, ".yo", "objects", hash)

	if err := ioutil.WriteFile(objectPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write object file: %w", err)
	}

	stagingPath := filepath.Join(repoPath, ".yo", "staging")
	f, err := os.OpenFile(stagingPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open staging area: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%s %s\n", hash, file)); err != nil {
		return fmt.Errorf("failed to write to staging area: %w", err)
	}

	return nil
}

func yoCommit(message string) error {
	repoPath, _ := os.Getwd()
	stagingPath := filepath.Join(repoPath, ".yo", "staging")

	stagingContent, err := ioutil.ReadFile(stagingPath)
	if err != nil {
		return fmt.Errorf("failed to read staging area: %w", err)
	}

	commitHash := hashObject(string(stagingContent) + message + time.Now().String())
	commitPath := filepath.Join(repoPath, ".yo", "objects", commitHash)

	if err := ioutil.WriteFile(commitPath, stagingContent, 0644); err != nil {
		return fmt.Errorf("failed to write commit object: %w", err)
	}

	logEntry := fmt.Sprintf("Commit: %s\nMessage: %s\nTime: %s\n\n", commitHash, message, time.Now().Format(time.RFC1123))
	logPath := filepath.Join(repoPath, ".yo", "logs", "commits")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	if _, err := logFile.WriteString(logEntry); err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}

	if err := os.Remove(stagingPath); err != nil {
		return fmt.Errorf("failed to clear staging area: %w", err)
	}

	return nil
}

func yoLog() (string, error) {
	repoPath, _ := os.Getwd()
	logPath := filepath.Join(repoPath, ".yo", "logs", "commits")

	logContent, err := ioutil.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to read log file: %w", err)
	}

	return string(logContent), nil
}

func hashObject(content string) string {
	// Create a SHA-1 hash of the content
	hasher := sha1.New()
	hasher.Write([]byte(content))
	return hex.EncodeToString(hasher.Sum(nil))
}

func startUI(message string) {
	p := tea.NewProgram(model{message: message})
	if err := p.Start(); err != nil {
		fmt.Printf("Error starting Bubble Tea program: %v\n", err)
		os.Exit(1)
	}
}
