package main

// Plik z tłumaczeniami (PL / EN).
// Zawiera wszystkie klucze używane przez GUI.

var Lang = map[string]map[string]string{
	"PL": {
		"chooseFile":      "  🗂️  Wybierz plik .h",
		"noFile":          "Brak wczytanego pliku",
		"loaded":          "Wczytano: ",
		"glyph":           "Znak",
		"editGlyph":       "✏️ Edytuj znak",
		"scale":           "Skala",
		"saveFont":        "💾 Zapisz cały font do .h",
		"save":            "📤  Zamknij / Pokaż w formacie C",
		"noData":          "Brak danych",
		"loadFirst":       "Najpierw wczytaj plik .h",
		"saved":           "Plik zapisany pomyślnie.",
		"close":           "Zamknij",
		"previewTitle":    "Znak %d w formacie C",
		"editWindowTitle": "✏️  Edytuj znak %d",
		// generowane wpisy
		"editedCharAscii": "// Znak edytowany: ASCII ",
		"generatedAuto":   "// Wygenerowano automatycznie — Font Preview v.%s\n",
		"charSize":        "// Rozmiar znaków: ",
		// błedy
		"saveError": "Błąd zapisu",
	},
	"EN": {
		"chooseFile":      "  🗂️  Choose .h file",
		"noFile":          "No file loaded",
		"loaded":          "Loaded: ",
		"glyph":           "Glyph",
		"editGlyph":       "✏️ Edit glyph",
		"scale":           "Scale",
		"saveFont":        "💾 Save entire font to .h",
		"save":            "📤  Close / Show in C format",
		"noData":          "No data",
		"loadFirst":       "Load .h file first",
		"saved":           "File saved successfully.",
		"close":           "Close",
		"previewTitle":    "Glyph %d in C format",
		"editWindowTitle": "✏️  Edit glyph %d",
		// generated text
		"editedCharAscii": "// Edited character: ASCII ",
		"generatedAuto":   "// Automatically generated — Font Preview v.%s\n",
		"charSize":        "// Character size: ",
		// errors
		"saveError": "Save error",
	},
}

// CurrentLang przechowuje aktualny język (domyślnie PL)
var CurrentLang = "PL"

// T zwraca tłumaczenie dla podanego klucza
func T(key string) string {
	if m, ok := Lang[CurrentLang]; ok {
		if v, ok2 := m[key]; ok2 {
			return v
		}
	}
	// fallback — jeśli brak klucza, zwracamy sam klucz, żeby widzieć błąd
	return key
}
