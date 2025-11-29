/* ============================================================================

    Font Preview & Editor Tool
    Wersja: 1.0.1
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
        - Dodane tłumaczenie PL/EN  -- plik i18n.go
        - Poprawki w układzie GUI
        - Poprawki Slidera ZOOM
        - Dodano tymczasową ikonkę ładowaną z resources/ plik png 256x256

=========================================================================== */

package main

import (
	//"bufio"
	//"fmt"
	"image/color"
	//"regexp"
	"strconv"
	//"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	//"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// -- Zmienne globalne -------------------------------------------------------------------

var versionApp = "1.0.3"      // wersja programu
var editWin fyne.Window       // okno edycji znaku (referencja globalna)
var editGrid *fyne.Container  // kontener z prostokątami w oknie edycji
var sliderInternalUpdate bool // Flaga blokująca pushUndo podczas aktualizacji sliderów
var xShift, yShift int        // globalne przesunięcia widoczne dla całego programu
var langBtn *widget.Button    // zmienna dla przycisku języka
var showGrid = true           // zmienna dla siatki w oknie edycji

var rects [][]*canvas.Rectangle // prostokąty reprezentujące piksele w edycji

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
		imgRaster.Refresh()
		updateEditorGrid(currentIndex, imgRaster)
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
			loadedFileLabel.SetText(T("loaded") + rc.URI().Name())
			defer func() { _ = rc.Close() }()
			nums, gw, gh, err := parseHeaderWithSize(rc)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			fontData = nums
			glyphW = gw
			glyphH = gh

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
		openEditWindow(currentIndex, imgRaster)
	})

	// Przycisk zapisu całego fontu
	saveAllBtn := widget.NewButton(T("saveFont"), func() {
		saveFontDialog(w)
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
		updateMainTexts(btn, loadedFileLabel, label, editBtn, scaleLabel, saveAllBtn, currentIndex, scale)
	})

	// Układ GUI głównego okna
	bottomBtns := container.NewVBox(
		saveAllBtn,
		langBtn,
	)

	content := container.NewBorder(
		nil,
		bottomBtns,
		nil,
		nil,
		container.NewVBox(
			btn,
			loadedFileLabel,
			label,
			slider,
			editBtn,
			scaleLabel,
			scaleSlider,
			container.NewCenter(imgRaster),
		),
	)

	w.SetContent(content)
	w.ShowAndRun()
}
