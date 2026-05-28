// Copyright 2019-2022 Graham Clark. All rights reserved.  Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

// Based heavily on vterm.py from urwid

package terminal

import (
	"fmt"

	"github.com/gcla/gowid"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/terminfo"
	log "github.com/sirupsen/logrus"
)

//======================================================================

type EventNotSupported struct {
	Event interface{}
}

var _ error = EventNotSupported{}

func (e EventNotSupported) Error() string {
	return fmt.Sprintf("Terminal input event %v of type %T not supported yet", e.Event, e.Event)
}

func pasteStart(ti *terminfo.Terminfo) []byte {
	return []byte("\x1b[200~")
}

func pasteEnd(ti *terminfo.Terminfo) []byte {
	return []byte("\x1b[201~")
}

func enablePaste(ti *terminfo.Terminfo) []byte {
	return []byte("\x1b[?2004h")

}

func disablePaste(ti *terminfo.Terminfo) []byte {
	return []byte("\x1b[?2004l")
}

// TCellEventToBytes converts TCell's representation of a terminal event to
// the string of bytes that would be the equivalent event according to the
// supplied Terminfo object. It returns a tuple of the byte slice
// representing the terminal event (if successful), and a bool (denoting
// success or failure). This function is used by the TerminalWidget. Its
// subprocess is connected to a tty controlled by gowid. Events from the
// user are parsed by gowid via TCell - they are then translated by this
// function before being written to the TerminalWidget subprocess's tty.
func TCellEventToBytes(ev interface{}, mouse IMouseSupport, last gowid.MouseState, paster IPaste, ti *terminfo.Terminfo) ([]byte, bool) {
	res := make([]byte, 0)
	res2 := false

	switch ev := ev.(type) {
	case *tcell.EventPaste:
		res2 = true
		if paster.PasteState() {
			// Already saw start
			res = append(res, pasteEnd(ti)...)
			paster.PasteState(false)
		} else {
			res = append(res, pasteStart(ti)...)
			paster.PasteState(true)
		}
	case *tcell.EventKey:
		if ev.Key() < ' ' {
			str := []rune{rune(ev.Key())}
			res = append(res, string(str)...)
			res2 = true
		} else {
			res2 = true
			switch ev.Key() {
			case tcell.KeyRune:
				str := []rune{ev.Rune()}
				res = append(res, string(str)...)
			case tcell.KeyCR:
				str := []rune{rune(tcell.KeyCR)}
				res = append(res, string(str)...)
			case tcell.KeyF1, tcell.KeyF2, tcell.KeyF3, tcell.KeyF4, tcell.KeyF5, tcell.KeyF6, tcell.KeyF7, tcell.KeyF8, tcell.KeyF9,
				tcell.KeyF10, tcell.KeyF11, tcell.KeyF12, tcell.KeyF13, tcell.KeyF14, tcell.KeyF15, tcell.KeyF16, tcell.KeyF17,
				tcell.KeyF18, tcell.KeyF19, tcell.KeyF20, tcell.KeyF21, tcell.KeyF22, tcell.KeyF23, tcell.KeyF24, tcell.KeyF25,
				tcell.KeyF26, tcell.KeyF27, tcell.KeyF28, tcell.KeyF29, tcell.KeyF30, tcell.KeyF31, tcell.KeyF32, tcell.KeyF33,
				tcell.KeyF34, tcell.KeyF35, tcell.KeyF36, tcell.KeyF37, tcell.KeyF38, tcell.KeyF39, tcell.KeyF40, tcell.KeyF41,
				tcell.KeyF42, tcell.KeyF43, tcell.KeyF44, tcell.KeyF45, tcell.KeyF46, tcell.KeyF47, tcell.KeyF48, tcell.KeyF49,
				tcell.KeyF50, tcell.KeyF51, tcell.KeyF52, tcell.KeyF53, tcell.KeyF54, tcell.KeyF55, tcell.KeyF56, tcell.KeyF57,
				tcell.KeyF58, tcell.KeyF59, tcell.KeyF60, tcell.KeyF61, tcell.KeyF62, tcell.KeyF63, tcell.KeyF64,
				tcell.KeyInsert, tcell.KeyDelete, tcell.KeyHome, tcell.KeyEnd, tcell.KeyHelp, tcell.KeyPgUp, tcell.KeyPgDn,
				tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight, tcell.KeyBacktab, tcell.KeyExit, tcell.KeyClear,
				tcell.KeyPrint, tcell.KeyCancel, tcell.KeyDEL, tcell.KeyBackspace:
				str := []rune{ev.Rune()}
				res = append(res, string(str)...)
			default:
				res2 = false
				panic(EventNotSupported{Event: ev})
			}
		}
	case *tcell.EventMouse:
		if mouse.MouseEnabled() {
			var data string

			btnind := 0
			switch ev.Buttons() {
			case tcell.Button1:
				btnind = 0
			case tcell.Button2:
				btnind = 1
			case tcell.Button3:
				btnind = 2
			case tcell.WheelUp:
				btnind = 64
			case tcell.WheelDown:
				btnind = 65
			}

			lastind := 0
			if last.LeftIsClicked() {
				lastind = 0
			} else if last.MiddleIsClicked() {
				lastind = 1
			} else if last.RightIsClicked() {
				lastind = 2
			}

			switch ev.Buttons() {
			case tcell.Button1, tcell.Button2, tcell.Button3, tcell.WheelUp, tcell.WheelDown:
				mx, my := ev.Position()
				btn := btnind
				if (last.LeftIsClicked() && (ev.Buttons() == tcell.Button1)) ||
					(last.MiddleIsClicked() && (ev.Buttons() == tcell.Button2)) ||
					(last.RightIsClicked() && (ev.Buttons() == tcell.Button3)) {
					// assume the mouse pointer has been moved with button down, a "drag"
					btn += 32
				}
				if mouse.MouseIsSgr() {
					data = fmt.Sprintf("\033[<%d;%d;%dM", btn, mx+1, my+1)
				} else {
					data = fmt.Sprintf("\033[M%c%c%c", btn+32, mx+33, my+33)
				}
				res = append(res, data...)
				res2 = true
			case tcell.ButtonNone:
				// TODO - how to report no press?
				mx, my := ev.Position()

				if last.LeftIsClicked() || last.MiddleIsClicked() || last.RightIsClicked() {
					// 0 means left mouse button, m means released
					if mouse.MouseIsSgr() {
						data = fmt.Sprintf("\033[<%d;%d;%dm", lastind, mx+1, my+1)
					} else if mouse.MouseReportAny() {
						data = fmt.Sprintf("\033[M%c%c%c", 35, mx+33, my+33)
					}
				} else if mouse.MouseReportAny() {
					if mouse.MouseIsSgr() {
						// +32 for motion, +3 for no button
						data = fmt.Sprintf("\033[<35;%d;%dm", mx+1, my+1)
					} else {
						data = fmt.Sprintf("\033[M%c%c%c", 35+32, mx+33, my+33)
					}
				}
				res = append(res, data...)
				res2 = true
			}
		}
	default:
		log.WithField("event", ev).Info("Event not implemented")
	}
	return res, res2
}

//======================================================================
// Local Variables:
// mode: Go
// fill-column: 110
// End:
