package main

import (
	"github.com/jroimartin/gocui"
)

func navigator(v *gocui.View, key gocui.Key, ch rune, mod gocui.Modifier) {
	// Rules for navigating the chat history
	switch {
	case key == gocui.KeyArrowUp:
		v.MoveCursor(0, -1, false)
	case key == gocui.KeyArrowDown:
		v.MoveCursor(0, 1, false)
	case key == gocui.KeyArrowLeft:
		v.MoveCursor(-1, 0, false)
	case key == gocui.KeyArrowRight:
		v.MoveCursor(1, 0, false)
	}
}

// Generate the UI and its rules
func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	// Log history, scrollable
	if v, err := g.SetView("logs", 1, 1, maxX/2-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView("logs"); err != nil {
			return err
		}
		v.Title = "Log history"
		v.Autoscroll = true
		v.Overwrite = false
		// We use the navigator as an editor, but the text displayed will not change
		v.Editor = gocui.EditorFunc(navigator)
		v.Editable = true
		v.Wrap = true
	}

	// Alert history, scrollable
	if v, err := g.SetView("alerts", maxX/2+1, 1, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		if _, err := g.SetCurrentView("alerts"); err != nil {
			return err
		}
		v.Title = "Alerts history"
		v.Autoscroll = true
		v.Overwrite = false
		// We use the navigator as an editor, but the text displayed will not change
		v.Editor = gocui.EditorFunc(navigator)
		v.Editable = true
		v.Wrap = true
	}
	// No error occured during any initialization
	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func update(g *gocui.Gui) error {
	return nil
}

func displayLine(g *gocui.Gui, viewName string, line string) {
	g.Update(
		func(g *gocui.Gui) error {
			if line[len(line)-1:] != "\n" {
				line = line + "\n"
			}
			byteMessage := []byte(line)
			originalView := g.CurrentView()
			v, err := g.SetCurrentView(viewName)
			if err != nil {
				return err
			}
			_, err = v.Write(byteMessage)
			if err != nil {
				return err
			}
			_, err = g.SetCurrentView(originalView.Name())
			if err != nil {
				return err
			}
			return nil
		})
}

func switchView(g *gocui.Gui, v *gocui.View) error {
	g.Update(
		func(g *gocui.Gui) error {
			currentView := g.CurrentView()
			currentView.Autoscroll = true
			originalViewName := currentView.Name()
			var newViewName string
			if originalViewName == "logs" {
				newViewName = "alerts"
			} else {
				newViewName = "logs"
			}
			_, err := g.SetCurrentView(newViewName)
			if err != nil {
				return err
			}
			return nil
		})
	return nil
}

// Initialize the keybindings that depend only on the gui
func initKeyBindings(g *gocui.Gui) error {
	// Ctrl-C leaves the application, whatever the focused view
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, switchView);
		err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyEnter, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			currentView := g.CurrentView()
			currentView.Autoscroll = false
			g.Update(update)
			return nil
		});
		err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, switchView);
		err != nil {
		return err
	}
	return nil
}
