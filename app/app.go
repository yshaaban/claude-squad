package app

import (
	"claude-squad/config"
	"claude-squad/keys"
	"claude-squad/log"
	"claude-squad/session"
	"claude-squad/ui"
	"claude-squad/ui/overlay"
	"claude-squad/web"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const GlobalInstanceLimit = 10

// Run is the main entrypoint into the application.
func Run(ctx context.Context, startOptions StartOptions) error {
	p := tea.NewProgram(
		newHome(ctx, startOptions),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(), // Mouse scroll
	)
	_, err := p.Run()
	return err
}

type state int

const (
	stateDefault state = iota
	// stateNew is the state when the user is creating a new instance.
	stateNew
	// statePrompt is the state when the user is entering a prompt.
	statePrompt
	// stateHelp is the state when a help screen is displayed.
	stateHelp
)

type home struct {
	ctx context.Context

	program string
	autoYes bool
	simpleMode bool

	// ui components
	list         *ui.List
	menu         *ui.Menu
	tabbedWindow *ui.TabbedWindow
	errBox       *ui.ErrBox
	// global spinner instance. we plumb this down to where it's needed
	spinner spinner.Model

	// storage is the interface for saving/loading data to/from the app's state
	storage *session.Storage
	// appConfig stores persistent application configuration
	appConfig *config.Config
	// appState stores persistent application state like seen help screens
	appState config.AppState
	
	// webServer holds the monitoring web server instance
	webServer *web.Server

	// state
	state state
	// newInstanceFinalizer is called when the state is stateNew and then you press enter.
	// It registers the new instance in the list after the instance has been started.
	newInstanceFinalizer func()

	// promptAfterName tracks if we should enter prompt mode after naming
	promptAfterName bool

	// textInputOverlay is the component for handling text input with state
	textInputOverlay *overlay.TextInputOverlay

	// textOverlay is the component for displaying text information
	textOverlay *overlay.TextOverlay

	// keySent is used to manage underlining menu items
	keySent bool
}

func newHome(ctx context.Context, startOptions StartOptions) *home {
	// Load application config
	appConfig := config.LoadConfig()

	// Load application state
	appState := config.LoadState()

	// Initialize storage
	storage, err := session.NewStorage(appState)
	if err != nil {
		// Return a properly error-handled home object
		errBox := ui.NewErrBox()
		errBox.SetError(fmt.Errorf("Failed to initialize storage: %w", err))
		return &home{
			errBox: errBox,
			ctx:    ctx,
		}
	}

	// Apply command line overrides to config
	if startOptions.WebServerEnabled {
		appConfig.WebServerEnabled = true
	}
	
	if startOptions.WebServerPort > 0 {
		appConfig.WebServerPort = startOptions.WebServerPort
	}

	h := &home{
		ctx:          ctx,
		spinner:      spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		menu:         ui.NewMenu(),
		tabbedWindow: ui.NewTabbedWindow(ui.NewPreviewPane(), ui.NewDiffPane()),
		errBox:       ui.NewErrBox(),
		storage:      storage,
		appConfig:    appConfig,
		program:      startOptions.Program,
		autoYes:      startOptions.AutoYes,
		simpleMode:   startOptions.SimpleMode,
		state:        stateDefault,
		appState:     appState,
	}
	h.list = ui.NewList(&h.spinner, startOptions.AutoYes)

	// Check if we're in simple mode
	if startOptions.SimpleMode {
		// Create a new instance to run in the current directory
		currentDir, err := os.Getwd()
		if err != nil {
			// Use the proper error handling mechanism
			h.errBox.SetError(fmt.Errorf("Failed to get current directory: %w", err))
			// Return the home object - the error will be displayed in the UI
			return h
		}
		
		// Check for existing simple mode instances in this directory
		instances, err := storage.LoadInstances()
		if err == nil {
			var staleInstances []string
			
			for _, instance := range instances {
				if instance.InPlace && filepath.Clean(instance.Path) == filepath.Clean(currentDir) {
					// Check if the instance's tmux session actually exists
					if instance.Started() && instance.TmuxAlive() {
						h.errBox.SetError(fmt.Errorf("A Simple Mode instance already exists for this directory. Please use that instance or run in a different directory."))
						
						// Add the existing instances to the list
						for _, existingInstance := range instances {
							h.list.AddInstance(existingInstance)()
							if startOptions.AutoYes {
								existingInstance.AutoYes = true
							}
						}
						
						return h
					} else {
						// This is a stale Simple Mode instance, mark it for removal
						staleInstances = append(staleInstances, instance.Title)
					}
				}
			}
			
			// Remove any stale Simple Mode instances for this directory
			for _, title := range staleInstances {
				log.InfoLog.Printf("Removing stale Simple Mode instance: %s", title)
				if err := storage.DeleteInstance(title); err != nil {
					log.ErrorLog.Printf("Error removing stale Simple Mode instance: %v", err)
				}
			}
		}
		
		// Create a default instance name based on timestamp
		instanceName := fmt.Sprintf("simple-%s", time.Now().Format("20060102-150405"))
		
		// Create a new instance that runs in-place (no worktree)
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:     instanceName,
			Path:      currentDir,
			Program:   startOptions.Program,
			AutoYes:   true,
			InPlace:   true,
		})
		if err != nil {
			// Use the proper error handling mechanism
			h.errBox.SetError(fmt.Errorf("Failed to create instance: %w", err))
			return h
		}
		
		// Start the instance immediately
		if err := instance.Start(true); err != nil {
			// Use the proper error handling mechanism
			h.errBox.SetError(fmt.Errorf("Failed to start instance: %w", err))
			return h
		}
		
		// Add instance to the list and select it
		h.list.AddInstance(instance)()
		h.list.SetSelectedInstance(0)
		instance.AutoYes = true

		// If web server is enabled in simple mode, automatically send an empty prompt
		// to create a Claude session immediately rather than showing the prompt dialog
		if startOptions.WebServerEnabled {
			log.InfoLog.Printf("Web server enabled in Simple Mode - sending empty prompt to start Claude session automatically")
			
			// Send an empty prompt to create the Claude session
			if err := instance.SendPrompt(""); err != nil {
				h.errBox.SetError(fmt.Errorf("Failed to send empty prompt: %w", err))
			}
			
			// Stay in default state since we've already sent the prompt
			h.state = stateDefault
			h.menu.SetState(ui.StateDefault)
		} else {
			// Standard simple mode behavior - show prompt dialog
			h.state = statePrompt
			h.menu.SetState(ui.StatePrompt)
			h.textInputOverlay = overlay.NewTextInputOverlay("Enter prompt", "")
		}
	} else {
		// Standard mode - load saved instances
		instances, err := storage.LoadInstances()
		if err != nil {
			// Use the proper error handling mechanism
			h.errBox.SetError(fmt.Errorf("Failed to load instances: %w", err))
			return h
		}

		// Add loaded instances to the list
		for _, instance := range instances {
			// Call the finalizer immediately.
			h.list.AddInstance(instance)()
			if startOptions.AutoYes {
				instance.AutoYes = true
			}
		}
	}
	
	// Start web server if enabled
	if appConfig.WebServerEnabled {
		log.InfoLog.Printf("Web server enabled, attempting to start on %s:%d", appConfig.WebServerHost, appConfig.WebServerPort)
		
		// Check if React UI is requested
		if startOptions.ReactUI {
			log.InfoLog.Printf("Using React frontend for web interface")
			if err := h.StartReactWebServer(); err != nil {
				h.errBox.SetError(fmt.Errorf("Failed to start React web server: %w", err))
			} else {
				// Update menu with web server info with React UI indicator
				h.menu.SetWebServerInfo(true, appConfig.WebServerHost, appConfig.WebServerPort)
				log.InfoLog.Printf("React web UI available at http://%s:%d/", 
					appConfig.WebServerHost, appConfig.WebServerPort)
				
				// Also log to standard error for visibility
				hostToDisplay := "localhost"
				if appConfig.WebServerHost != "" {
					hostToDisplay = appConfig.WebServerHost
				}
				fmt.Printf("\nReact web UI available: http://%s:%d/\n", 
					hostToDisplay, 
					appConfig.WebServerPort)
			}
		} else {
			// Standard web server
			if err := h.StartWebServer(); err != nil {
				h.errBox.SetError(fmt.Errorf("Failed to start web server: %w", err))
			} else {
				// Update menu with web server info
				h.menu.SetWebServerInfo(true, appConfig.WebServerHost, appConfig.WebServerPort)
			}
		}
	}

	return h
}

// updateHandleWindowSizeEvent sets the sizes of the components.
// The components will try to render inside their bounds.
func (m *home) updateHandleWindowSizeEvent(msg tea.WindowSizeMsg) {
	var listWidth int
	
	// In simple mode, list takes minimal width (10%)
	if m.simpleMode {
		listWidth = int(float32(msg.Width) * 0.1)
	} else {
		// Standard mode - list takes 30% of width
		listWidth = int(float32(msg.Width) * 0.3)
	}
	
	tabsWidth := msg.Width - listWidth

	// Menu takes 10% of height, list and window take 90%
	contentHeight := int(float32(msg.Height) * 0.9)
	menuHeight := msg.Height - contentHeight - 1     // minus 1 for error box
	m.errBox.SetSize(int(float32(msg.Width)*0.9), 1) // error box takes 1 row

	m.tabbedWindow.SetSize(tabsWidth, contentHeight)
	m.list.SetSize(listWidth, contentHeight)

	if m.textInputOverlay != nil {
		m.textInputOverlay.SetSize(int(float32(msg.Width)*0.6), int(float32(msg.Height)*0.4))
	}
	if m.textOverlay != nil {
		m.textOverlay.SetWidth(int(float32(msg.Width) * 0.6))
	}

	previewWidth, previewHeight := m.tabbedWindow.GetPreviewSize()
	if err := m.list.SetSessionPreviewSize(previewWidth, previewHeight); err != nil {
		log.ErrorLog.Print(err)
	}
	m.menu.SetSize(msg.Width, menuHeight)
}

func (m *home) Init() tea.Cmd {
	// Upon starting, we want to start the spinner. Whenever we get a spinner.TickMsg, we
	// update the spinner, which sends a new spinner.TickMsg. I think this lasts forever lol.
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			time.Sleep(100 * time.Millisecond) // Initial quick update
			// Subsequent updates will be slower to reduce load
			return previewTickMsg{isInitial: true}
		},
		tickUpdateMetadataCmd,
	)
}

func (m *home) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case hideErrMsg:
		m.errBox.Clear()
	case previewTickMsg:
		cmd := m.instanceChanged()
		// Reduce polling frequency after initial fast updates
		delay := 500 * time.Millisecond // Slower general polling rate
		if msg.isInitial {
			delay = 250 * time.Millisecond // A bit faster for the first few ticks
		}
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				time.Sleep(delay)
				return previewTickMsg{isInitial: false}
			},
		)
	case keyupMsg:
		m.menu.ClearKeydown()
		return m, nil
	case tickUpdateMetadataMessage:
		for _, instance := range m.list.GetInstances() {
			if !instance.Started() || instance.Paused() {
				continue
			}
			// Capture content once, then use it for updates
			// This relies on changes in Instance.HasUpdated to accept cached content
			currentContent, err := instance.Preview() // This still happens, but HasUpdated will be cheaper
			if err != nil {
				log.WarningLog.Printf("could not get preview for metadata update %s: %v", instance.Title, err)
				continue
			}
			updated, prompt := instance.HasUpdated(currentContent)
			if updated {
				instance.SetStatus(session.Running)
			} else if !prompt { // If not updated and not a prompt, it's ready
				instance.SetStatus(session.Ready)
			}
			if prompt && instance.AutoYes { // AutoYes logic for prompts
				instance.TapEnter()
			}
			if err := instance.UpdateDiffStats(); err != nil {
				log.WarningLog.Printf("could not update diff stats: %v", err)
			}
		}
		return m, tickUpdateMetadataCmd
	case tea.MouseMsg:
		// Handle mouse wheel scrolling in the diff view
		if m.tabbedWindow.IsInDiffTab() {
			if msg.Action == tea.MouseActionPress {
				switch msg.Button {
				case tea.MouseButtonWheelUp:
					m.tabbedWindow.ScrollUp()
					return m, m.instanceChanged()
				case tea.MouseButtonWheelDown:
					m.tabbedWindow.ScrollDown()
					return m, m.instanceChanged()
				}
			}
		}
		return m, nil
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.updateHandleWindowSizeEvent(msg)
		return m, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *home) handleQuit() (tea.Model, tea.Cmd) {
	// Save instances before quitting
	if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
		return m, m.handleError(err)
	}
	
	// When in Simple Mode, we only want to kill that specific Claude instance
	// and remove it from storage so it doesn't appear in future sessions
	if m.simpleMode {
		selected := m.list.GetSelectedInstance()
		if selected != nil && selected.Started() && !selected.Paused() && selected.InPlace {
			log.InfoLog.Printf("Terminating Simple Mode instance: %s", selected.Title)
			
			// Kill the instance
			if err := selected.Kill(); err != nil {
				log.ErrorLog.Printf("Error terminating instance %s: %v", selected.Title, err)
			}
			
			// Remove it from storage as well
			if err := m.storage.DeleteInstance(selected.Title); err != nil {
				log.ErrorLog.Printf("Error removing Simple Mode instance from storage: %v", err)
			} else {
				log.InfoLog.Printf("Removed Simple Mode instance %s from storage", selected.Title)
			}
		}
	}
	
	// Shutdown web server if running
	m.StopWebServer()
	
	// Quit the application
	return m, tea.Quit
}

func (m *home) handleMenuHighlighting(msg tea.KeyMsg) (cmd tea.Cmd, returnEarly bool) {
	// Handle menu highlighting when you press a button. We intercept it here and immediately return to
	// update the ui while re-sending the keypress. Then, on the next call to this, we actually handle the keypress.
	if m.keySent {
		m.keySent = false
		return nil, false
	}
	if m.state == statePrompt || m.state == stateHelp {
		return nil, false
	}
	// If it's in the global keymap, we should try to highlight it.
	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return nil, false
	}

	if m.list.GetSelectedInstance() != nil && m.list.GetSelectedInstance().Paused() && name == keys.KeyEnter {
		return nil, false
	}
	if name == keys.KeyShiftDown || name == keys.KeyShiftUp {
		return nil, false
	}

	// Skip the menu highlighting if the key is not in the map or we are using the shift up and down keys.
	// TODO: cleanup: when you press enter on stateNew, we use keys.KeySubmitName. We should unify the keymap.
	if name == keys.KeyEnter && m.state == stateNew {
		name = keys.KeySubmitName
	}
	m.keySent = true
	return tea.Batch(
		func() tea.Msg { return msg },
		m.keydownCallback(name)), true
}

func (m *home) handleKeyPress(msg tea.KeyMsg) (mod tea.Model, cmd tea.Cmd) {
	cmd, returnEarly := m.handleMenuHighlighting(msg)
	if returnEarly {
		return m, cmd
	}

	if m.state == stateHelp {
		return m.handleHelpState(msg)
	}

	if m.state == stateNew {
		// Handle quit commands first. Don't handle q because the user might want to type that.
		if msg.String() == "ctrl+c" {
			m.state = stateDefault
			m.promptAfterName = false
			m.list.Kill()
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		}

		instance := m.list.GetInstances()[m.list.NumInstances()-1]
		switch msg.Type {
		// Start the instance (enable previews etc) and go back to the main menu state.
		case tea.KeyEnter:
			if len(instance.Title) == 0 {
				return m, m.handleError(fmt.Errorf("title cannot be empty"))
			}

			if err := instance.Start(true); err != nil {
				m.list.Kill()
				m.state = stateDefault
				return m, m.handleError(err)
			}
			// Save after adding new instance
			if err := m.storage.SaveInstances(m.list.GetInstances()); err != nil {
				return m, m.handleError(err)
			}
			// Instance added successfully, call the finalizer.
			m.newInstanceFinalizer()
			if m.autoYes {
				instance.AutoYes = true
			}

			m.newInstanceFinalizer()
			m.state = stateDefault
			if m.promptAfterName {
				m.state = statePrompt
				m.menu.SetState(ui.StatePrompt)
				// Initialize the text input overlay
				m.textInputOverlay = overlay.NewTextInputOverlay("Enter prompt", "")
				m.promptAfterName = false
			} else {
				m.menu.SetState(ui.StateDefault)
				m.showHelpScreen(helpTypeInstanceStart, nil)
			}

			return m, tea.Batch(tea.WindowSize(), m.instanceChanged())
		case tea.KeyRunes:
			if len(instance.Title) >= 32 {
				return m, m.handleError(fmt.Errorf("title cannot be longer than 32 characters"))
			}
			if err := instance.SetTitle(instance.Title + string(msg.Runes)); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyBackspace:
			if len(instance.Title) == 0 {
				return m, nil
			}
			if err := instance.SetTitle(instance.Title[:len(instance.Title)-1]); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeySpace:
			if err := instance.SetTitle(instance.Title + " "); err != nil {
				return m, m.handleError(err)
			}
		case tea.KeyEsc:
			m.list.Kill()
			m.state = stateDefault
			m.instanceChanged()

			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					return nil
				},
			)
		default:
		}
		return m, nil
	} else if m.state == statePrompt {
		// Use the new TextInputOverlay component to handle all key events
		shouldClose := m.textInputOverlay.HandleKeyPress(msg)

		// Check if the form was submitted or canceled
		if shouldClose {
			if m.textInputOverlay.IsSubmitted() {
				// Form was submitted, process the input
				selected := m.list.GetSelectedInstance()
				if selected == nil {
					return m, nil
				}
				if err := selected.SendPrompt(m.textInputOverlay.GetValue()); err != nil {
					return m, m.handleError(err)
				}
			}

			// Close the overlay and reset state
			m.textInputOverlay = nil
			m.state = stateDefault
			return m, tea.Sequence(
				tea.WindowSize(),
				func() tea.Msg {
					m.menu.SetState(ui.StateDefault)
					m.showHelpScreen(helpTypeInstanceStart, nil)
					return nil
				},
			)
		}

		return m, nil
	}

	// Handle quit commands first
	if msg.String() == "ctrl+c" || msg.String() == "q" {
		return m.handleQuit()
	}

	name, ok := keys.GlobalKeyStringsMap[msg.String()]
	if !ok {
		return m, nil
	}

	switch name {
	case keys.KeyHelp:
		return m.showHelpScreen(helpTypeGeneral, nil)
	case keys.KeyPrompt:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:   "",
			Path:    ".",
			Program: m.program,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)
		m.promptAfterName = true

		return m, nil
	case keys.KeyNew:
		if m.list.NumInstances() >= GlobalInstanceLimit {
			return m, m.handleError(
				fmt.Errorf("you can't create more than %d instances", GlobalInstanceLimit))
		}
		instance, err := session.NewInstance(session.InstanceOptions{
			Title:   "",
			Path:    ".",
			Program: m.program,
		})
		if err != nil {
			return m, m.handleError(err)
		}

		m.newInstanceFinalizer = m.list.AddInstance(instance)
		m.list.SetSelectedInstance(m.list.NumInstances() - 1)
		m.state = stateNew
		m.menu.SetState(ui.StateNewInstance)

		return m, nil
	case keys.KeyUp:
		m.list.Up()
		return m, m.instanceChanged()
	case keys.KeyDown:
		m.list.Down()
		return m, m.instanceChanged()
	case keys.KeyShiftUp:
		if m.tabbedWindow.IsInDiffTab() {
			m.tabbedWindow.ScrollUp()
		}
		return m, m.instanceChanged()
	case keys.KeyShiftDown:
		if m.tabbedWindow.IsInDiffTab() {
			m.tabbedWindow.ScrollDown()
		}
		return m, m.instanceChanged()
	case keys.KeyTab:
		m.tabbedWindow.Toggle()
		m.menu.SetInDiffTab(m.tabbedWindow.IsInDiffTab())
		return m, m.instanceChanged()
	case keys.KeyKill:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		worktree, err := selected.GetGitWorktree()
		if err != nil {
			return m, m.handleError(err)
		}

		checkedOut, err := worktree.IsBranchCheckedOut()
		if err != nil {
			return m, m.handleError(err)
		}

		if checkedOut {
			return m, m.handleError(fmt.Errorf("instance %s is currently checked out", selected.Title))
		}

		// Delete from storage first
		if err := m.storage.DeleteInstance(selected.Title); err != nil {
			return m, m.handleError(err)
		}

		// Then kill the instance
		m.list.Kill()
		return m, m.instanceChanged()
	case keys.KeySubmit:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Default commit message with timestamp
		commitMsg := fmt.Sprintf("[claudesquad] update from '%s' on %s", selected.Title, time.Now().Format(time.RFC822))
		
		// Handle Simple Mode differently - use direct git commands
		if selected.InPlace {
			// Execute git commands directly on the current directory
			
			// First check if there are any changes to commit
			gitStatusCmd := exec.Command("git", "status", "--porcelain")
			gitStatusCmd.Dir = selected.Path
			statusOutput, err := gitStatusCmd.Output()
			if err != nil {
				return m, m.handleError(fmt.Errorf("failed to get git status: %w", err))
			}
			
			// If no changes, show message and return
			if len(statusOutput) == 0 {
				return m, m.handleError(fmt.Errorf("no changes to commit"))
			}
			
			// Add all changes
			gitAddCmd := exec.Command("git", "add", ".")
			gitAddCmd.Dir = selected.Path
			if err := gitAddCmd.Run(); err != nil {
				return m, m.handleError(fmt.Errorf("failed to stage changes: %w", err))
			}
			
			// Commit changes
			gitCommitCmd := exec.Command("git", "commit", "-m", commitMsg)
			gitCommitCmd.Dir = selected.Path
			if err := gitCommitCmd.Run(); err != nil {
				return m, m.handleError(fmt.Errorf("failed to commit changes: %w", err))
			}
			
			// Push changes
			gitPushCmd := exec.Command("git", "push")
			gitPushCmd.Dir = selected.Path
			if err := gitPushCmd.Run(); err != nil {
				return m, m.handleError(fmt.Errorf("failed to push changes: %w", err))
			}
			
			// Show success message
			m.errBox.SetInfo("Changes committed and pushed successfully")
			return m, func() tea.Msg {
				time.Sleep(3 * time.Second)
				return hideErrMsg{}
			}
		} else {
			// Standard mode - use worktree
			worktree, err := selected.GetGitWorktree()
			if err != nil {
				return m, m.handleError(err)
			}
			if err = worktree.PushChanges(commitMsg, true); err != nil {
				return m, m.handleError(err)
			}
		}

		return m, nil
	case keys.KeyCheckout:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}

		// Show help screen before pausing
		m.showHelpScreen(helpTypeInstanceCheckout, func() {
			if err := selected.Pause(); err != nil {
				m.handleError(err)
			}
			m.instanceChanged()
		})
		return m, nil
	case keys.KeyResume:
		selected := m.list.GetSelectedInstance()
		if selected == nil {
			return m, nil
		}
		if err := selected.Resume(); err != nil {
			return m, m.handleError(err)
		}
		return m, tea.WindowSize()
	case keys.KeyEnter:
		if m.list.NumInstances() == 0 {
			return m, nil
		}
		selected := m.list.GetSelectedInstance()
		if selected == nil || selected.Paused() || !selected.TmuxAlive() {
			return m, nil
		}
		// Show help screen before attaching
		m.showHelpScreen(helpTypeInstanceAttach, func() {
			ch, err := m.list.Attach()
			if err != nil {
				m.handleError(err)
				return
			}
			<-ch
			m.state = stateDefault
		})
		return m, nil
	default:
		return m, nil
	}
}

// instanceChanged updates the preview pane, menu, and diff pane based on the selected instance. It returns an error
// Cmd if there was any error.
func (m *home) instanceChanged() tea.Cmd {
	// selected may be nil
	selected := m.list.GetSelectedInstance()

	m.tabbedWindow.UpdateDiff(selected)
	// Update menu with current instance
	m.menu.SetInstance(selected)

	// If there's no selected instance, we don't need to update the preview.
	if err := m.tabbedWindow.UpdatePreview(selected); err != nil {
		return m.handleError(err)
	}
	return nil
}

type keyupMsg struct{}

// keydownCallback clears the menu option highlighting after 500ms.
func (m *home) keydownCallback(name keys.KeyName) tea.Cmd {
	m.menu.Keydown(name)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(500 * time.Millisecond):
		}

		return keyupMsg{}
	}
}

// hideErrMsg implements tea.Msg and clears the error text from the screen.
type hideErrMsg struct{}

// previewTickMsg implements tea.Msg and triggers a preview update
type previewTickMsg struct{
	isInitial bool // Flag to allow faster initial updates
}

type tickUpdateMetadataMessage struct{}

// tickUpdateMetadataCmd is the callback to update the metadata of the instances every 500ms. Note that we iterate
// overall the instances and capture their output. It's a pretty expensive operation. Let's do it 2x a second only.
var tickUpdateMetadataCmd = func() tea.Msg {
	time.Sleep(500 * time.Millisecond)
	return tickUpdateMetadataMessage{}
}

// handleError handles all errors which get bubbled up to the app. sets the error message. We return a callback tea.Cmd that returns a hideErrMsg message
// which clears the error message after 3 seconds.
func (m *home) handleError(err error) tea.Cmd {
	log.ErrorLog.Printf("%v", err)
	m.errBox.SetError(err)
	return func() tea.Msg {
		select {
		case <-m.ctx.Done():
		case <-time.After(3 * time.Second):
		}

		return hideErrMsg{}
	}
}

func (m *home) View() string {
	listWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.list.String())
	previewWithPadding := lipgloss.NewStyle().PaddingTop(1).Render(m.tabbedWindow.String())
	listAndPreview := lipgloss.JoinHorizontal(lipgloss.Top, listWithPadding, previewWithPadding)

	mainView := lipgloss.JoinVertical(
		lipgloss.Center,
		listAndPreview,
		m.menu.String(),
		m.errBox.String(),
	)

	if m.state == statePrompt {
		if m.textInputOverlay == nil {
			log.ErrorLog.Printf("text input overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textInputOverlay.Render(), mainView, true, true)
	} else if m.state == stateHelp {
		if m.textOverlay == nil {
			log.ErrorLog.Printf("text overlay is nil")
		}
		return overlay.PlaceOverlay(0, 0, m.textOverlay.Render(), mainView, true, true)
	}

	return mainView
}