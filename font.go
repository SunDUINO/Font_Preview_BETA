/* ============================================================================

    Font Handling
    Funkcje do wczytywania fontów .h oraz zapisu całej tablicy
    – parseHeaderWithSize, saveFontDialog

=========================================================================== */

package main

import (
	"bufio"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// Globalne dane fontu
var fontData []uint16
var glyphW, glyphH int

// parseHeaderWithSize odczytuje font z pliku .h i wykrywa wymiary znaków
func parseHeaderWithSize(r fyne.URIReadCloser) ([]uint16, int, int, error) {
	sc := bufio.NewScanner(r)
	hexRE := regexp.MustCompile(`0x[0-9A-Fa-f]+`)
	nameRE := regexp.MustCompile(`(?i)uint16_t\s+(\w+)`) // nazwa tablicy

	var nums []uint16
	var gw, gh int

	for sc.Scan() {
		line := sc.Text()

		// Wykrycie wymiarów z nazwy tablicy np. "ALGER_16x16"
		if gw == 0 || gh == 0 {
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
							gw = w
							gh = h
						}
					}
				}
			}
		}

		matches := hexRE.FindAllString(line, -1)
		for _, m := range matches {
			v, err := strconv.ParseUint(m, 0, 16)
			if err != nil {
				return nil, 0, 0, err
			}
			nums = append(nums, uint16(v))
		}
	}

	return nums, gw, gh, sc.Err()
}

// Wywoływane przy kliknięciu "Save Font"
func saveFontDialog(w fyne.Window) {
	if len(fontData) == 0 {
		dialog.ShowInformation(T("noData"), T("loadFirst"), w)
		return
	}

	dialog.ShowFileSave(func(uc fyne.URIWriteCloser, _ error) {
		if uc == nil {
			return
		}
		defer func() { _ = uc.Close() }()

		var sb strings.Builder

		// Nagłówek
		sb.WriteString(fmt.Sprintf(T("generatedAuto"), versionApp))
		sb.WriteString(T("charSize"))
		sb.WriteString(fmt.Sprintf("%dx%d\n\n", glyphW, glyphH))

		// Nazwa tablicy
		sb.WriteString("const uint16_t FONT_" + strconv.Itoa(glyphW) + "x" + strconv.Itoa(glyphH) + "[] = {\n")

		total := len(fontData) / glyphH
		for i := 0; i < total; i++ {
			sb.WriteString("   ")
			for y := 0; y < glyphH; y++ {
				row := fontData[i*glyphH+y]
				sb.WriteString(fmt.Sprintf("0x%04X,", row))
			}
			ch := i + 32
			if ch >= 32 && ch <= 126 {
				sb.WriteString(fmt.Sprintf("  // '%c'", rune(ch)))
			} else {
				sb.WriteString("  //")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("};\n")

		if _, err := uc.Write([]byte(sb.String())); err != nil {
			fmt.Println(T("saveError")+": ", err)
		}
		dialog.ShowInformation(T("saved"), T("saved"), w)
	}, w)
}
