package main

// A simple program demonstrating the text input component from the Bubbles
// component library.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

	initialProgram := tea.NewProgram(initialModel())

	m, err := initialProgram.Run()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	if m, ok := m.(inputModel); ok && m.homeserver != "" {
		menuProgram := tea.NewProgram(choicesModel{homeserver: m.homeserver})
		_, err := menuProgram.Run()
		if err != nil {
			fmt.Println("Oh no:", err)
			os.Exit(1)
		}
	}
}

// Choices Model
type choicesModel struct {
	homeserver string
	cursor     int
	choice     string
	result     string
	err        string
}

var choices = []string{"Version", "Help"}

func (m choicesModel) View() string {
	s := strings.Builder{}
	s.WriteString("Homeserver: " + m.homeserver + "\n\n")
	s.WriteString("Actions available: \n\n")
	for i := 0; i < len(choices); i++ {
		if m.cursor == i {
			s.WriteString("(•) ")
		} else {
			s.WriteString("( ) ")
		}
		s.WriteString(choices[i])
		s.WriteString("\n")
	}
	s.WriteString("\nResult: " + m.result + " \n")
	if m.err != "" {
		s.WriteString("\nError: " + m.err + " \n")
	}
	s.WriteString("\n(press q to quit)\n")

	return s.String()
}

func (m choicesModel) Init() tea.Cmd {
	return nil
}

func (m choicesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "enter":
			// Send the choice on the channel and exit.
			m.choice = choices[m.cursor]
			if choices[m.cursor] == "Version" {
				result, err := checkServer(m.homeserver)
				if err != nil {
					m.err = err.Error()
				}
				m.result = result
			}
			if choices[m.cursor] == "Help" {
				m.result = "Choose one option"
				return m, nil
			}
			return m, tea.Quit

		case "down", "j":
			m.cursor++
			if m.cursor >= len(choices) {
				m.cursor = 0
			}

		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(choices) - 1
			}
		}
	}

	return m, nil
}

// Input Model
type (
	errMsg error
)

type inputModel struct {
	homeserver string
	textInput  textinput.Model
	err        error
}

func initialModel() inputModel {
	ti := textinput.New()
	ti.Placeholder = "my-chat.server.com"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return inputModel{
		homeserver: "",
		textInput:  ti,
		err:        nil,
	}
}

func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			m.homeserver = m.textInput.Value()
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

func (m inputModel) View() string {
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
	var result ResponseVersion
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.ServerVersion, nil
}
