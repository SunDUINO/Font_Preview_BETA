/* ============================================================================

    Funkcje Undo i REDO
    Wszystkie funkcje związane z oknem edycji pojedynczego znaku
    – umozliwiają cofanie i ponawianie zmian w edycji pikseli siatki oraz
      przesunięć sliderem X i Y

=========================================================================== */

package main

// Stosy UNDO / REDO
var undoStack []GlyphState
var redoStack []GlyphState

type GlyphState struct {
	Data    []uint16
	OffsetX int
	OffsetY int
}

// Zapisuje aktualny stan glifu i offsetów
func snapshotState(index, h int) GlyphState {
	snap := make([]uint16, h)
	copy(snap, fontData[index*h:index*h+h])
	return GlyphState{
		Data:    snap,
		OffsetX: xShift,
		OffsetY: yShift,
	}
}

// Przywraca stan glifu i offsetów
func restoreState(index, h int, state GlyphState) {
	copy(fontData[index*h:index*h+h], state.Data)
	xShift = state.OffsetX
	yShift = state.OffsetY
}

// zapisywanie stanu aktualnie edytowanego glifu do stosu UNDO
func pushUndo(index int) {
	if glyphH == 0 {
		return
	}
	undoStack = append(undoStack, snapshotState(index, glyphH))
	redoStack = nil
}

// Funkcja Undo
func undo(index int) bool {
	if len(undoStack) == 0 {
		return false
	}
	last := undoStack[len(undoStack)-1]
	undoStack = undoStack[:len(undoStack)-1]
	redoStack = append(redoStack, snapshotState(index, glyphH))
	restoreState(index, glyphH, last)
	return true
}

// Funkcja redo
func redo(index int) bool {
	if len(redoStack) == 0 {
		return false
	}
	last := redoStack[len(redoStack)-1]
	redoStack = redoStack[:len(redoStack)-1]
	undoStack = append(undoStack, snapshotState(index, glyphH))
	restoreState(index, glyphH, last)
	return true
}
