/* ============================================================================

    Font Preview & Editor Tool
    Wersja: 0.0.13
    Autor: Lothar Team / SunRiver
           Lothar Team / Gufim
    Data: listopad 2025

    Opis:
    ---------------------------------------------------------------------------
    Ten program umożliwia:
      • wczytywanie plików czcionek w formacie C (.h) opartych o uint16_t,
      • automatyczne wykrywanie wymiarów znaków z nazwy tablicy (np. 16x16),
      • podgląd znaków w formie siatki bitmapowej,
      • edycję pojedynczego znaku w osobnym oknie,
      • modyfikację bitów poprzez siatkę prostokątów (klik – zmiana koloru),
      • skalowanie podglądu znaku,
      • przesuwanie znaku w osi X/Y (shift) w oknie edycji,
      • aktualizację w czasie rzeczywistym widoczną w głównym podglądzie,
      • generowanie fragmentu kodu C dla edytowanego glifu,
      • zapisywanie całej zmodyfikowanej tablicy jako pliku .h.

    Technologie:
      • GUI zbudowane w Fyne (Go)
      • Render bitmapy poprzez canvas.NewRasterWithPixels
      • Manipulacja tablicą uint16 odzwierciedlającą poziome wiersze glifa
      • Edycja siatki z wykorzystaniem kontenera bez layoutu (Manual layout)

    Uwagi:
      • Każdy wiersz znaku to jeden uint16 – bity odpowiadają pikselom.
      • Edycja zapisuje zmiany bezpośrednio do fontData[].
      • Obsługuje dowolny rozmiar czcionki (np. 5x8, 8x16, 16x16, 32x32…)
      • Zmiany są widoczne natychmiast w obu oknach.

    Nowe:
      • 26.11.2022
        - Dodane tłumaczenie PL/EN
        - Poprawki w układzie GUI
        - Poprawki Slidera ZOOM
        - Dodano tymczasową ikonkę ładowaną z resources/ plik png 256x256

=========================================================================== */

package main

import (
	"bufio"
	"fmt"
	"image/color"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// -- Zmienne globalne -------------------------------------------------------------------
var versionApp = "0.0.14" // wersja priogramu

var fontData []uint16           // tablica z danymi fontu
var glyphW, glyphH int          // wymiary pojedynczego znaku
var editWin fyne.Window         // okno edycji znaku (referencja globalna)
var editGrid *fyne.Container    // kontener z prostokątami w oknie edycji
var rects [][]*canvas.Rectangle // prostokąty reprezentujące piksele w edycji
var xShift, yShift int          // globalne przesunięcia widoczne dla całego programu
var langBtn *widget.Button      // zmienna dla przycisku języka

// Główna funkcja programu  ----------------------------------------------------------------
// Dodano ikonke
func main() {

	a := app.NewWithID("com.lothar-team.fontpreview") // identyfikator programu
	w := a.NewWindow(" Font Preview v." + versionApp) // nazwa programu + nr wersji
	w.Resize(fyne.NewSize(400, 750))                  // ustawienie początkowego rozmiaru
	w.SetFixedSize(true)                              // blokada zmiany rozmiaru okna

	// Załaduj ikonę z pliku
	icon, err := fyne.LoadResourceFromPath("resources/AB256.png")
	if err != nil {
		println("Błąd ładowania ikony:", err.Error())
	} else {
		w.SetIcon(icon)
	}

	currentIndex := 0 // aktualny indeks znaku
	scale := 7        // początkowa skala powiększenia

	loadedFileLabel := widget.NewLabel(T("noFile")) // wyświetlanie nazwy otwartego pliku

	// Raster dynamiczny do wyświetlania znaku
	imgRaster := canvas.NewRasterWithPixels(func(x, y, wR, hR int) color.Color {
		if len(fontData) == 0 || glyphW == 0 || glyphH == 0 {
			return color.White
		}

		gx := x / scale
		gy := y / scale

		// sprawdzamy czy piksel mieści się w polu rysowania
		if gx < 0 || gy < 0 || gx >= glyphW || gy >= glyphH {
			return color.White
		}

		// --- NOWE: uwzględnienie przesunięcia ---
		adjX := gx - xShift
		adjY := gy - yShift

		if adjX < 0 || adjY < 0 || adjX >= glyphW || adjY >= glyphH {
			return color.White
		}

		row := fontData[currentIndex*glyphH+adjY]
		bit := (row >> (glyphW - 1 - adjX)) & 1
		if bit != 0 {
			return color.Black
		}
		return color.White
	})

	imgRaster.SetMinSize(fyne.NewSize(float32(16*scale), float32(16*scale)))

	// Etykieta pokazująca numer indeksu aktualnego znaku z tablicy
	label := widget.NewLabel(T("glyph") + ": 0")

	// Slider wyboru znaku
	slider := widget.NewSlider(0, 0)
	slider.Step = 1
	slider.OnChanged = func(val float64) {
		currentIndex = int(val)
		label.SetText(T("glyph") + ": " + strconv.Itoa(currentIndex))
		imgRaster.Refresh() // odświeżenie podglądu
		// Jeśli okno edycji jest otwarte, zaktualizuj jego prostokąty
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
		}
	}

	// Slider zmiany skali
	scaleSlider := widget.NewSlider(1, 14)
	scaleSlider.Value = float64(scale)
	scaleLabel := widget.NewLabel(T("scale") + ": " + strconv.Itoa(scale))
	scaleSlider.OnChanged = func(val float64) {
		scale = int(val)
		scaleLabel.SetText(T("scale") + ": " + strconv.Itoa(scale))
		if glyphW > 0 && glyphH > 0 {
			imgRaster.SetMinSize(fyne.NewSize(float32(glyphW*scale), float32(glyphH*scale)))
			imgRaster.Refresh()
		}
	}

	// Przycisk wczytywania pliku .h
	btn := widget.NewButton(T("chooseFile"), func() {
		dialog.ShowFileOpen(func(rc fyne.URIReadCloser, _ error) {
			if rc == nil {
				return
			}
			// -- USTAWIENIE NAZWY WCZYTANEGO PLIKU
			loadedFileLabel.SetText(T("loaded") + rc.URI().Name())

			defer func() { _ = rc.Close() }() // jawne ignorowanie błędu
			nums, gw, gh, err := parseHeaderWithSize(rc)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			fontData = nums
			glyphW = gw
			glyphH = gh

			// Aktualizacja slidera
			slider.Max = float64(len(fontData)/glyphH - 1)
			currentIndex = 0
			slider.Value = 0
			label.SetText(T("glyph") + ": 0")

			imgRaster.SetMinSize(fyne.NewSize(float32(glyphW*scale), float32(glyphH*scale)))
			imgRaster.Refresh()
		}, w)
	})

	// Przycisk edycji znaku
	editBtn := widget.NewButton(T("editGlyph"), func() {
		if len(fontData) == 0 || glyphW == 0 || glyphH == 0 {
			return
		}

		// Tworzymy okno edycji aktualnego znaku
		// Dodano ikonke
		editWin = fyne.CurrentApp().NewWindow(fmt.Sprintf(T("editWindowTitle"), currentIndex))

		pixelSize := 20.0
		gridWidth := float32(float64(glyphW) * pixelSize)
		gridHeight := float32(float64(glyphH) * pixelSize)

		// Kontener bez layoutu
		editGrid = container.NewWithoutLayout()
		rects = make([][]*canvas.Rectangle, glyphH)
		for y := 0; y < glyphH; y++ {
			rects[y] = make([]*canvas.Rectangle, glyphW)
			for x := 0; x < glyphW; x++ {
				xx, yy := x, y
				rect := canvas.NewRectangle(color.White)
				rect.StrokeColor = color.Gray{Y: 128}
				rect.StrokeWidth = 1
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
						row := fontData[currentIndex*glyphH+yy]
						row ^= 1 << (glyphW - 1 - xx)
						fontData[currentIndex*glyphH+yy] = row
						// Aktualizacja prostokąta w edycji
						if (row>>(glyphW-1-xx))&1 != 0 {
							rects[yy][xx].FillColor = color.Black
						} else {
							rects[yy][xx].FillColor = color.White
						}
						rects[yy][xx].Refresh()
						imgRaster.Refresh() // odświeżenie głównego podglądu
					}
				}(xx, yy))
				btn.Importance = widget.LowImportance
				btn.Resize(fyne.NewSize(float32(pixelSize), float32(pixelSize)))
				btn.Move(fyne.NewPos(float32(xx)*float32(pixelSize), float32(yy)*float32(pixelSize)))
				editGrid.Add(btn)
			}
		}

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
					tmp[newY] = shiftRow(fontData[currentIndex*glyphH+y], xShift, glyphW)
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
			imgRaster.Refresh() // odświeżenie głównego podglądu
		}

		// --- Slidery do przsuwania znaku :
		// Suwak X – przesuwanie znaku w poziomie
		xSlider := widget.NewSlider(float64(-(glyphW - 1)), float64(glyphW-1))
		xSlider.Value = 0
		xSlider.Step = 1
		xSlider.OnChanged = func(val float64) {
			xShift = int(val)
			refreshGrid()
		}

		// Suwak Y – przesuwanie znaku w pionie
		ySlider := widget.NewSlider(float64(-(glyphH - 1)), float64(glyphH-1))
		ySlider.Value = 0
		ySlider.Step = 1
		ySlider.OnChanged = func(val float64) {
			yShift = int(val)
			refreshGrid()
		}

		// Przycisk zapisu i pokazania znaku w formacie C
		// Dodano ikonke
		saveBtn := widget.NewButton(T("save"), func() {

			// Zastosowanie przesunięć X i Y do fontData
			if xShift != 0 || yShift != 0 {
				// przygotuj tymczasowy bufor
				tmp := make([]uint16, glyphH)

				// przesuwanie w pionie
				for y := 0; y < glyphH; y++ {
					newY := y + yShift
					if newY >= 0 && newY < glyphH {
						tmp[newY] = shiftRow(fontData[currentIndex*glyphH+y], xShift, glyphW)
					}
				}

				// przepisanie przesuniętych danych do fontData
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

		// Umieszczenie gridu i przycisków z suwakami w oknie edycji
		content := container.NewBorder(nil, container.NewVBox(saveBtn, xSlider, ySlider), nil, nil, editGrid)
		editWin.SetContent(content)
		editWin.Resize(fyne.NewSize(gridWidth+2, gridHeight+100))
		editWin.Show()
	})

	saveAllBtn := widget.NewButton(T("saveFont"), func() {
		if len(fontData) == 0 {
			dialog.ShowInformation(T("noData"), T("loadFirst"), w)
			return
		}

		dialog.ShowFileSave(func(uc fyne.URIWriteCloser, _ error) {
			if uc == nil {
				return
			}

			defer func() {
				_ = uc.Close()
			}()

			var sb strings.Builder

			// Nagłówek
			sb.WriteString(fmt.Sprintf(T("generatedAuto"), versionApp))
			sb.WriteString(T("charSize"))
			sb.WriteString(fmt.Sprintf("%dx%d\n\n", glyphW, glyphH))

			// Nazwa tablicy
			sb.WriteString("const uint16_t FONT_" + strconv.Itoa(glyphW) + "x" + strconv.Itoa(glyphH) + "[] = {\n")

			// Zawartość tablicy
			total := len(fontData) / glyphH
			for i := 0; i < total; i++ {
				sb.WriteString("   ")

				for y := 0; y < glyphH; y++ {
					row := fontData[i*glyphH+y]
					sb.WriteString(fmt.Sprintf("0x%04X,", row))
				}

				// komentarz z symbolem ASCII
				ch := i + 32
				if ch >= 32 && ch <= 126 {
					sb.WriteString(fmt.Sprintf("  // '%c'", rune(ch)))
				} else {
					sb.WriteString("  //")
				}
				sb.WriteString("\n")
			}

			sb.WriteString("};\n")

			// Zapis
			if _, err := uc.Write([]byte(sb.String())); err != nil {
				fmt.Println(T("saveError")+": ", err)
			}

			dialog.ShowInformation(T("saved"), T("saved"), w)
		}, w)
	})

	// ---> przycisk zmiany jezyka PL/EN ---
	langBtn = widget.NewButton("🇬🇧", func() {
		if CurrentLang == "PL" {
			CurrentLang = "EN"
			langBtn.SetText("🇵🇱")
		} else {
			CurrentLang = "PL"
			langBtn.SetText("🇬🇧")
		}

		// Aktualizacja wszystkich tekstów GUI
		btn.SetText(T("chooseFile"))
		loadedFileLabel.SetText(T("noFile"))
		label.SetText(T("glyph") + ": " + strconv.Itoa(currentIndex))
		editBtn.SetText(T("editGlyph"))
		scaleLabel.SetText(T("scale") + ": " + strconv.Itoa(scale))
		saveAllBtn.SetText(T("saveFont"))

	})

	// Układ GUI głównego okna
	bottomBtns := container.NewVBox(
		saveAllBtn,
		langBtn,
	)
	// Zmiana kontenera
	content := container.NewBorder(
		nil,        // nic u góry
		bottomBtns, // przyklejone do dołu
		nil,        // brak po lewej
		nil,        // brak po prawej
		container.NewVBox(
			btn, // Wczytaj plik
			loadedFileLabel,
			label,
			slider,
			editBtn,
			scaleLabel,
			scaleSlider,
			container.NewCenter(imgRaster), // glif wyśrodkowany
		),
	)

	w.SetContent(content)
	w.ShowAndRun()
}

// parseHeaderWithSize odczytuje font z pliku .h i wykrywa wymiary znaków
func parseHeaderWithSize(r fyne.URIReadCloser) ([]uint16, int, int, error) {
	sc := bufio.NewScanner(r)
	hexRE := regexp.MustCompile(`0x[0-9A-Fa-f]+`)
	nameRE := regexp.MustCompile(`(?i)uint16_t\s+(\w+)`) // nazwa tablicy

	var nums []uint16
	var glyphW, glyphH int

	for sc.Scan() {
		line := sc.Text()

		// Wykrycie wymiarów z nazwy tablicy np. "ALGER_16x16"
		if glyphW == 0 || glyphH == 0 {
			match := nameRE.FindStringSubmatch(line)
			if len(match) > 1 {
				name := match[1]
				parts := strings.Split(name, "_")
				if len(parts) > 1 {
					sizePart := parts[len(parts)-1]
					dims := strings.Split(sizePart, "x")
					if len(dims) == 2 {
						w, err1 := strconv.Atoi(dims[0])
						h, err2 := strconv.Atoi(dims[1])
						if err1 == nil && err2 == nil {
							glyphW = w
							glyphH = h
						}
					}
				}
			}
		}

		// Parsowanie liczb hex do tablicy
		matches := hexRE.FindAllString(line, -1)
		for _, m := range matches {
			v, err := strconv.ParseUint(m, 0, 16)
			if err != nil {
				return nil, 0, 0, err
			}
			nums = append(nums, uint16(v))
		}
	}

	return nums, glyphW, glyphH, sc.Err()
}
