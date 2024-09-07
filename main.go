package main

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close() // nolint:errcheck
	}

	p := tea.NewProgram(initialModel())

	m, err := p.Run()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if m, ok := m.(model); ok && m.version != "" {
		fmt.Printf("\n---\nVersion: %s!\n", m.version)
	}

	fmt.Println("Exiting")
}

type (
	errMsg error
)

type model struct {
	version   string
	textInput textinput.Model
	err       error
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "my-chat.server.com"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		version:   "",
		textInput: ti,
		err:       nil,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			version, err := checkServer(m.textInput.Value())
			if err != nil {
				return m, tea.Quit
			}
			m.version = version
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return fmt.Sprintf(
		"Provide Synapse Homeserver URL\n\n%s\n%s",
		m.textInput.View(),
		"(press esc to quit)",
	) + "\n"
}

// ResponseVersion holds data from server_version request
type ResponseVersion struct {
	ServerVersion string `json:"server_version"`
}

func checkServer(inputURL string) (string, error) {
	c := &http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := c.Get("https://" + inputURL + "/_synapse/admin/v1/server_version")
	if err != nil {
		return "", err
	}
	defer res.Body.Close() // nolint:errcheck
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	log.Println(string(body))
	var result ResponseVersion
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	log.Println(result)
	return result.ServerVersion, nil
}
