package coders

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/coding-hui/ai-terminal/internal/cli/options"
	"github.com/coding-hui/ai-terminal/internal/git"
	"github.com/coding-hui/ai-terminal/internal/llm"
	"github.com/coding-hui/ai-terminal/internal/ui"
	"github.com/coding-hui/ai-terminal/internal/util/display"
)

var program *tea.Program

type State struct {
	error      error
	buffer     string
	querying   bool
	confirming bool
}

// AutoCoder is a auto generate coders user interface.
type AutoCoder struct {
	state State

	command      *command
	gitRepo      *git.Command
	codeBasePath string
	absFileNames map[string]struct{}

	history        *ui.History
	checkpointChan chan Checkpoint
	checkpoints    []Checkpoint

	cfg       *options.Config
	llmEngine *llm.Engine
}

func StartAutCoder() error {
	coder := NewAutoCoder()
	program = tea.NewProgram(
		coder,
		// tea.WithAltScreen(),       // use the full size of the terminal in its "alternate screen buffer"
		tea.WithMouseCellMotion(), // turn on mouse support so we can track the mouse wheel
	)
	if _, err := program.Run(); err != nil {
		fmt.Println("Error running auto chat program:", err)
		os.Exit(1)
	}
	return nil
}

func NewAutoCoder() *AutoCoder {
	var err error
	g := git.New()
	cfg := options.NewConfig()

	autoCoder := &AutoCoder{
		state: State{
			error:  nil,
			buffer: "",
		},
		gitRepo:        g,
		cfg:            cfg,
		checkpoints:    []Checkpoint{},
		history:        ui.NewHistory(),
		absFileNames:   map[string]struct{}{},
		checkpointChan: make(chan Checkpoint),
	}

	autoCoder.llmEngine, err = llm.NewLLMEngine(llm.ChatEngineMode, cfg)
	if err != nil {
		display.FatalErr(err)
	}

	root, err := g.GitDir()
	if err != nil {
		display.FatalErr(err)
	}

	autoCoder.codeBasePath = filepath.Dir(root)
	autoCoder.command = newCommand(autoCoder)

	return autoCoder
}

func (a *AutoCoder) Init() tea.Cmd {
	return tea.Sequence(
		tea.ClearScreen,
		tea.Println(components.renderer.RenderContent(components.renderer.RenderWelcomeMessage(a.cfg.System.GetUsername()))),
		textarea.Blink,
		a.statusTickCmd(),
	)
}

func (a *AutoCoder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds       []tea.Cmd
		promptCmd  tea.Cmd
		spinnerCmd tea.Cmd
		confirmCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case spinner.TickMsg:
		if a.state.querying {
			components.spinner, spinnerCmd = components.spinner.Update(msg)
			cmds = append(
				cmds,
				spinnerCmd,
			)
		}

	case tea.WindowSizeMsg:
		components.width = msg.Width
		components.height = msg.Height
		components.prompt.SetWidth(msg.Width)
		if a.state.confirming {
			components.confirm.SetWidth(msg.Width)
			components.confirm.SetHeight(msg.Height - components.prompt.Height())
			components.confirm.GotoBottom()
		}

	case Checkpoint:
		if len(a.checkpoints) <= 0 || a.checkpoints[len(a.checkpoints)-1].Desc != msg.Desc {
			a.checkpoints = append(a.checkpoints, msg)
		}
		cmds = append(
			cmds,
			a.statusTickCmd(),
			components.spinner.Tick,
		)

	case WaitFormUserConfirm:
		components.confirm, confirmCmd = components.confirm.Update(msg)
		return a, tea.Sequence(
			confirmCmd,
			textarea.Blink,
		)

	case tea.KeyMsg:
		switch msg.Type {
		// quit
		case tea.KeyCtrlC:
			return a, tea.Quit

		// help
		case tea.KeyCtrlH:
			if !a.state.querying && !a.state.confirming {
				components.prompt.SetValue("")
				components.prompt, promptCmd = components.prompt.Update(msg)
				cmds = append(
					cmds,
					promptCmd,
					tea.Println(components.renderer.RenderContent(components.renderer.RenderHelpMessage())),
					textarea.Blink,
				)
			}

		// history
		case tea.KeyUp, tea.KeyDown:
			if !a.state.querying && !a.state.confirming {
				var input *string
				if msg.Type == tea.KeyUp {
					input = a.history.GetPrevious()
				} else {
					input = a.history.GetNext()
				}
				if input != nil {
					components.prompt.SetValue(*input)
					components.prompt, promptCmd = components.prompt.Update(msg)
					cmds = append(
						cmds,
						promptCmd,
					)
				}
			}

		// handle user input
		case tea.KeyEnter:
			input := components.prompt.GetValue()
			if !a.state.querying && !a.state.confirming && input != "" {
				a.state.buffer = ""
				a.checkpoints = make([]Checkpoint, 0)
				a.history.Add(input)
				inputPrint := components.prompt.AsString()
				components.prompt.SetValue("")
				components.prompt.Blur()
				components.prompt, promptCmd = components.prompt.Update(msg)
				cmds = append(
					cmds,
					promptCmd,
					tea.Println(inputPrint),
					a.command.run(input),
					a.command.awaitChatCompleted(),
				)
				components.prompt.Focus()
			}

		// clear
		case tea.KeyCtrlL:
			if !a.state.querying && !a.state.confirming {
				a.checkpoints = make([]Checkpoint, 0)
				components.prompt.SetValue("")
				components.prompt, promptCmd = components.prompt.Update(msg)
				cmds = append(
					cmds,
					promptCmd,
					tea.ClearScreen,
					textarea.Blink,
				)
			}

		// reset
		case tea.KeyCtrlR:
			if !a.state.querying && !a.state.confirming {
				a.reset()
				components.prompt.SetValue("")
				components.prompt, promptCmd = components.prompt.Update(msg)
				cmds = append(
					cmds,
					promptCmd,
					tea.ClearScreen,
					textarea.Blink,
				)
			}

		default:
			if a.state.confirming && components.confirm != nil {
				components.confirm, confirmCmd = components.confirm.Update(msg)
				return a, tea.Sequence(
					confirmCmd,
					textarea.Blink,
				)
			}
			components.prompt.Focus()
			components.prompt, promptCmd = components.prompt.Update(msg)
			cmds = append(
				cmds,
				promptCmd,
				textarea.Blink,
			)
		}

	// engine chat stream feedback
	case llm.EngineChatStreamOutput:
		if msg.IsLast() {
			output := components.renderer.RenderContent(a.state.buffer)
			components.prompt.Focus()
			return a, tea.Sequence(
				tea.Println(output),
				textarea.Blink,
			)
		} else {
			return a, a.command.awaitChatCompleted()
		}

	case error:
		a.state.error = msg
		return a, nil
	}

	return a, tea.Batch(cmds...)
}

func (a *AutoCoder) View() string {
	if components.width == 0 || components.height == 0 {
		return "Initializing..."
	}

	started := len(a.checkpoints) > 0
	done := started && a.checkpoints[len(a.checkpoints)-1].Done

	if a.state.confirming && components.confirm != nil {
		return components.confirm.View()
	}

	if started && a.checkpoints[len(a.checkpoints)-1].Error != nil {
		return fmt.Sprintf("\n%s\n\n%s\n",
			components.renderer.RenderError(fmt.Sprintf("%s", a.checkpoints[len(a.checkpoints)-1].Error)),
			components.prompt.View(),
		)
	}

	if started {
		doneMsg := ""
		for _, s := range a.checkpoints[:len(a.checkpoints)-1] {
			icon := checkpointIcon(s.Type)
			switch s.Type {
			case StatusLoading:
				if !done {
					doneMsg += icon + s.Desc + "\n"
				}
			case StatusSuccess:
				doneMsg += icon + components.renderer.RenderSuccess(s.Desc) + "\n"
			case StatusWarning:
				doneMsg += icon + components.renderer.RenderWarning(s.Desc) + "\n"
			default:
				doneMsg += icon + s.Desc + "\n"
			}
		}
		if !done {
			if len(a.state.buffer) > 0 {
				return components.renderer.RenderContent(a.state.buffer)
			}
			return components.spinner.ViewWithMessage(doneMsg, a.checkpoints[len(a.checkpoints)-1].Desc)
		}
		if len(doneMsg) > 0 {
			return fmt.Sprintf("\n%s\n%s",
				doneMsg,
				components.prompt.View(),
			)
		}
	}

	return components.prompt.View()
}

func (a *AutoCoder) reset() {
	a.checkpoints = make([]Checkpoint, 0)
	a.history.Reset()
	a.absFileNames = make(map[string]struct{})
	a.state.buffer = ""
}
