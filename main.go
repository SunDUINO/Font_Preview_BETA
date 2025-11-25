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

func main() {
	// Tworzymy aplikację z unikalnym ID (wymóg Fyne do użycia Preferences)
	a := app.NewWithID("com.lothar-team.fontpreview")
	w := a.NewWindow("Font Preview v.0.0.beta ")

	var fontData []uint16  // wczytane dane fontu
	var glyphW, glyphH int // wymiary pojedynczego znaku
	currentIndex := 0      // aktualny znak do wyświetlenia
	scale := 8             // początkowa skala powiększenia

	// Raster dynamiczny – generuje obraz znaków na bieżąco
	imgRaster := canvas.NewRasterWithPixels(func(x, y, wR, hR int) color.Color {
		if len(fontData) == 0 || glyphW == 0 || glyphH == 0 {
			return color.White
		}

		// Przeliczamy współrzędne piksela do współrzędnych w macierzy znaku
		gx := x / scale
		gy := y / scale
		if gx >= glyphW || gy >= glyphH {
			return color.White
		}

		row := fontData[currentIndex*glyphH+gy]
		bit := (row >> (glyphW - 1 - gx)) & 1
		if bit != 0 {
			return color.Black
		}
		return color.White
	})

	// Minimalny rozmiar obrazu, zostanie nadpisany po wczytaniu fontu
	imgRaster.SetMinSize(fyne.NewSize(float32(16*scale), float32(16*scale)))

	// Etykieta pokazująca numer aktualnego znaku
	label := widget.NewLabel("Znak: 0")

	// Slider do wyboru znaku
	slider := widget.NewSlider(0, 0)
	slider.Step = 1
	slider.OnChanged = func(val float64) {
		currentIndex = int(val)
		label.SetText("Znak: " + strconv.Itoa(currentIndex))
		imgRaster.Refresh() // odświeżamy raster po zmianie znaku
	}

	// Slider do zmiany skali powiększenia
	scaleSlider := widget.NewSlider(1, 32) // skala od 1 do 32
	scaleSlider.Value = float64(scale)
	scaleLabel := widget.NewLabel("Skala: " + strconv.Itoa(scale))
	scaleSlider.OnChanged = func(val float64) {
		scale = int(val)
		scaleLabel.SetText("Skala: " + strconv.Itoa(scale))
		if glyphW > 0 && glyphH > 0 {
			imgRaster.SetMinSize(fyne.NewSize(float32(glyphW*scale), float32(glyphH*scale)))
			imgRaster.Refresh()
		}
	}

	// Przycisk do wczytania pliku .h z fontem
	// Dodano ikonke
	btn := widget.NewButton("  🗂️  Wybierz plik .h", func() {
		dialog.ShowFileOpen(func(rc fyne.URIReadCloser, _ error) {
			if rc == nil {
				return
			}

			// Zamknięcie pliku po zakończeniu funkcji z obsługą błędu
			defer func() {
				if err := rc.Close(); err != nil {
					fmt.Println("Błąd przy zamykaniu pliku:", err)
				}
			}()

			// Wczytanie fontu i wykrycie wymiarów znaków
			nums, gw, gh, err := parseHeaderWithSize(rc)
			if err != nil {
				dialog.ShowError(err, w) // w tym miejscu 'w' to okno Fyne
				return
			}
			fontData = nums
			glyphW = gw
			glyphH = gh

			// Aktualizacja slidera do wyboru znaków
			slider.Max = float64(len(fontData)/glyphH - 1)
			currentIndex = 0
			slider.Value = 0
			label.SetText("Znak: 0")

			// Ustawienie minimalnego rozmiaru rastra według wymiarów i skali
			imgRaster.SetMinSize(fyne.NewSize(float32(glyphW*scale), float32(glyphH*scale)))
			imgRaster.Refresh()
		}, w)
	})

	// Układ GUI
	content := container.NewVBox(
		btn,
		label,
		slider,
		scaleLabel,
		scaleSlider,
		imgRaster,
	)

	w.SetContent(content)
	w.Resize(fyne.NewSize(400, 600))
	w.ShowAndRun()
}

// parseHeaderWithSize odczytuje font z pliku .h i automatycznie wykrywa wymiary znaków
func parseHeaderWithSize(r fyne.URIReadCloser) ([]uint16, int, int, error) {
	sc := bufio.NewScanner(r)
	hexRE := regexp.MustCompile(`0x[0-9A-Fa-f]+`)
	nameRE := regexp.MustCompile(`(?i)uint16_t\s+(\w+)`) // nazwa tablicy

	var nums []uint16
	var glyphW, glyphH int

	for sc.Scan() {
		line := sc.Text()

		// Wykrycie wymiarów z nazwy tablicy, np. "ALGER_16x16"
		if glyphW == 0 || glyphH == 0 {
			match := nameRE.FindStringSubmatch(line)
			if len(match) > 1 {
				name := match[1] // np. ALGER_16x16
				parts := strings.Split(name, "_")
				if len(parts) > 1 {
					sizePart := parts[len(parts)-1] // "16x16"
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

		// Parsowanie liczb hex do tablicy uint16
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
