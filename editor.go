/* ============================================================================

    Edytor Glifu
    Wszystkie funkcje związane z oknem edycji pojedynczego znaku
    – prostokąty pikseli, kliknięcia, przesunięcia, UNDO/REDO, zapisywanie glifu

=========================================================================== */

package main

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// Otwiera okno edycji glifu
func openEditWindow(currentIndex int, imgRaster *canvas.Raster) {

	if len(fontData) == 0 || glyphW == 0 || glyphH == 0 {
		return
	}

	editWin = fyne.CurrentApp().NewWindow(fmt.Sprintf(T("editWindowTitle"), currentIndex))

	pixelSize := 20.0
	gridWidth := float32(float64(glyphW) * pixelSize)
	gridHeight := float32(float64(glyphH) * pixelSize)

	editGrid = container.NewWithoutLayout()
	rects = make([][]*canvas.Rectangle, glyphH)

	for y := 0; y < glyphH; y++ {
		rects[y] = make([]*canvas.Rectangle, glyphW)
		for x := 0; x < glyphW; x++ {
			xx, yy := x, y
			rect := canvas.NewRectangle(color.White)
			if showGrid {
				rect.StrokeColor = color.Gray{Y: 128}
				rect.StrokeWidth = 1
			} else {
				rect.StrokeWidth = 0
			}
			rect.Resize(fyne.NewSize(float32(pixelSize), float32(pixelSize)))
			rect.Move(fyne.NewPos(float32(xx)*float32(pixelSize), float32(yy)*float32(pixelSize)))

			// inicjalizacja koloru
			row := fontData[currentIndex*glyphH+yy]
			if (row>>(glyphW-1-xx))&1 != 0 {
				rect.FillColor = color.Black
			}
			rects[yy][xx] = rect
			editGrid.Add(rect)

			// Klikalny przycisk nad prostokątem
			btn := widget.NewButton("", func(xx, yy int) func() {
				return func() {
					pushUndo(currentIndex)
					row := fontData[currentIndex*glyphH+yy]
					row ^= 1 << (glyphW - 1 - xx)
					fontData[currentIndex*glyphH+yy] = row
					if (row>>(glyphW-1-xx))&1 != 0 {
						rects[yy][xx].FillColor = color.Black
					} else {
						rects[yy][xx].FillColor = color.White
					}
					rects[yy][xx].Refresh()
					imgRaster.Refresh()
				}
			}(xx, yy))
			btn.Importance = widget.LowImportance
			btn.Resize(fyne.NewSize(float32(pixelSize), float32(pixelSize)))
			btn.Move(fyne.NewPos(float32(xx)*float32(pixelSize), float32(yy)*float32(pixelSize)))
			editGrid.Add(btn)
		}
	}

	// Checkbox - pokaż siatkę
	gridCheck := widget.NewCheck(T("showGrid"), func(val bool) {
		showGrid = val
		for y := 0; y < glyphH; y++ {
			for x := 0; x < glyphW; x++ {
				if showGrid {
					rects[y][x].StrokeWidth = 1
					rects[y][x].StrokeColor = color.Gray{Y: 128}
				} else {
					rects[y][x].StrokeWidth = 0
				}
				rects[y][x].Refresh()
			}
		}
	})
	gridCheck.SetChecked(showGrid)

	// Funkcja pomocnicza do przesunięcia bitów w wierszu
	shiftRow := func(row uint16, shift int, width int) uint16 {
		if shift > 0 {
			return (row << shift) & ((1 << width) - 1)
		} else if shift < 0 {
			return row >> (-shift)
		}
		return row
	}

	// Funkcja odświeżająca prostokąty w edycji z uwzględnieniem przesunięcia
	refreshGrid := func() {
		tmp := make([]uint16, glyphH)
		for y := 0; y < glyphH; y++ {
			newY := y + yShift
			if newY >= 0 && newY < glyphH {
				tmp[newY] = shiftRow(fontData[currentIndex*glyphH+y], -xShift, glyphW)
			}
		}
		for y := 0; y < glyphH; y++ {
			row := tmp[y]
			for x := 0; x < glyphW; x++ {
				if (row>>(glyphW-1-x))&1 != 0 {
					rects[y][x].FillColor = color.Black
				} else {
					rects[y][x].FillColor = color.White
				}
				rects[y][x].Refresh()
			}
		}
		imgRaster.Refresh()
	}

	// Strzałki dla suwaków
	leftArrow := canvas.NewText("◀️", color.Black)
	leftArrow.Alignment = fyne.TextAlignCenter
	leftArrow.Resize(fyne.NewSize(32, 32))
	rightArrow := canvas.NewText("▶️", color.Black)
	rightArrow.Alignment = fyne.TextAlignCenter
	rightArrow.Resize(fyne.NewSize(32, 32))
	upArrow := canvas.NewText("🔼", color.Black)
	upArrow.Alignment = fyne.TextAlignCenter
	upArrow.Resize(fyne.NewSize(32, 32))
	downArrow := canvas.NewText("🔽", color.Black)
	downArrow.Alignment = fyne.TextAlignCenter
	downArrow.Resize(fyne.NewSize(32, 32))

	// Slider X
	xSlider := widget.NewSlider(float64(-(glyphW - 1)), float64(glyphW-1))
	xSlider.Value = 0
	xSlider.Step = 1
	xSlider.OnChanged = func(val float64) {
		if sliderInternalUpdate {
			return
		}
		pushUndo(currentIndex)
		xShift = int(val)
		refreshGrid()
	}

	// Slider Y
	ySlider := widget.NewSlider(float64(-(glyphH - 1)), float64(glyphH-1))
	ySlider.Value = 0
	ySlider.Step = 1
	ySlider.OnChanged = func(val float64) {
		if sliderInternalUpdate {
			return
		}
		pushUndo(currentIndex)
		yShift = int(val)
		refreshGrid()
	}

	// Dodanie strzałek
	xSliderWithArrows := container.NewBorder(
		nil,
		nil,
		leftArrow,
		rightArrow,
		xSlider,
	)
	ySliderWithArrows := container.NewBorder(
		nil,
		nil,
		upArrow,
		downArrow,
		ySlider,
	)

	// Przycisk UNDO/REDO
	undoBtn := widget.NewButton(T("undo"), func() {
		if undo(currentIndex) {
			sliderInternalUpdate = true
			xSlider.SetValue(float64(xShift))
			ySlider.SetValue(float64(yShift))
			sliderInternalUpdate = false
			refreshGrid()
		}
	})
	redoBtn := widget.NewButton(T("redo"), func() {
		if redo(currentIndex) {
			sliderInternalUpdate = true
			xSlider.SetValue(float64(xShift))
			ySlider.SetValue(float64(yShift))
			sliderInternalUpdate = false
			refreshGrid()
		}
	})

	// Przycisk zapisu glifu
	saveBtn := widget.NewButton(T("save"), func() {
		if xShift != 0 || yShift != 0 {
			tmp := make([]uint16, glyphH)
			for y := 0; y < glyphH; y++ {
				newY := y + yShift
				if newY >= 0 && newY < glyphH {
					tmp[newY] = shiftRow(fontData[currentIndex*glyphH+y], xShift, glyphW)
				}
			}
			for y := 0; y < glyphH; y++ {
				fontData[currentIndex*glyphH+y] = tmp[y]
			}
		}

		var sb strings.Builder
		sb.WriteString(T("editedCharAscii"))
		sb.WriteString(fmt.Sprintf("'%c'\n", currentIndex+32))
		for y := 0; y < glyphH; y++ {
			row := fontData[currentIndex*glyphH+y]
			sb.WriteString(fmt.Sprintf("0x%04X", row))
			if y < glyphH-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString(fmt.Sprintf(", // '%c'\n", currentIndex+32))

		previewWin := fyne.CurrentApp().NewWindow(fmt.Sprintf(T("previewTitle"), currentIndex))
		previewEntry := widget.NewMultiLineEntry()
		previewEntry.SetText(sb.String())
		previewEntry.Wrapping = fyne.TextWrapBreak
		previewWin.SetContent(container.NewVBox(
			previewEntry,
			widget.NewButton(T("close"), func() { previewWin.Close() }),
		))
		previewWin.Resize(fyne.NewSize(900, 120))
		previewWin.Show()

		editWin.Close()
		editWin = nil
		imgRaster.Refresh()
	})

	content := container.NewBorder(
		nil,
		container.NewVBox(
			xSliderWithArrows,
			ySliderWithArrows,
			saveBtn,
			container.NewHBox(undoBtn, redoBtn, gridCheck),
		),
		nil,
		nil,
		editGrid,
	)
	editWin.SetContent(content)
	editWin.Resize(fyne.NewSize(gridWidth+8, gridHeight+200))
	editWin.Show()
}

// Aktualizacja prostokątów w edytorze po zmianie znaku w głównym oknie
func updateEditorGrid(currentIndex int, imgRaster *canvas.Raster) {
	if editWin != nil && editGrid != nil && len(rects) == glyphH {
		for y := 0; y < glyphH; y++ {
			for x := 0; x < glyphW; x++ {
				row := fontData[currentIndex*glyphH+y]
				if (row>>(glyphW-1-x))&1 != 0 {
					rects[y][x].FillColor = color.Black
				} else {
					rects[y][x].FillColor = color.White
				}
				rects[y][x].Refresh()
			}
		}
		imgRaster.Refresh()
	}
}

// Aktualizacja tekstów w GUI po zmianie języka
func updateMainTexts(btn, loadedFileLabel, label, editBtn, scaleLabel, saveAllBtn interface{}, currentIndex, scale int) {
	btn.(*widget.Button).SetText(T("chooseFile"))
	loadedFileLabel.(*widget.Label).SetText(T("noFile"))
	label.(*widget.Label).SetText(T("glyph") + ": " + strconv.Itoa(currentIndex))
	editBtn.(*widget.Button).SetText(T("editGlyph"))
	scaleLabel.(*widget.Label).SetText(T("scale") + ": " + strconv.Itoa(scale))
	saveAllBtn.(*widget.Button).SetText(T("saveFont"))
}
