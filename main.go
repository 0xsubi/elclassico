// elclassico — generates index.html from menu.csv
//
// CSV columns: Section, Menu item, Price, Sort order
//
// Usage:
//   elclassico [--csv menu.csv] [--out index.html]

package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ─── data model ──────────────────────────────────────────────────────────────

type Item struct {
	Name  string
	Price string
}

type Section struct {
	ID    string
	Name  string
	Items []Item
}

// ─── CSV loading ─────────────────────────────────────────────────────────────

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func loadCSV(path string) ([]Section, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.TrimLeadingSpace = true

	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("cannot read CSV header: %w", err)
	}

	// Find column indices (case-insensitive)
	colIdx := map[string]int{
		"section":    -1,
		"menu item":  -1,
		"price":      -1,
		"sort order": -1,
	}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, ok := colIdx[key]; ok {
			colIdx[key] = i
		}
	}
	for k, v := range colIdx {
		if v == -1 {
			return nil, fmt.Errorf("required column %q not found in CSV header", k)
		}
	}

	ci := colIdx["section"]
	mi := colIdx["menu item"]
	pi := colIdx["price"]
	si := colIdx["sort order"]

	type rawItem struct {
		name      string
		price     string
		sortOrder float64 // NaN = no order given
		rowNum    int
	}

	sectionOrder := []string{}             // IDs in first-appearance order
	sectionNames := map[string]string{}    // id -> display name
	sectionItems := map[string][]rawItem{} // id -> rows

	rowNum := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("CSV read error: %w", err)
		}
		rowNum++

		get := func(idx int) string {
			if idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}

		sectionName := get(ci)
		itemName := get(mi)

		// Skip fully empty rows
		if sectionName == "" && itemName == "" {
			continue
		}
		if sectionName == "" {
			continue
		}

		id := slugify(sectionName)
		if _, seen := sectionItems[id]; !seen {
			sectionOrder = append(sectionOrder, id)
			sectionNames[id] = sectionName // first occurrence sets display name
			sectionItems[id] = nil
		}

		if itemName == "" {
			continue // section row with no item (e.g. placeholder rows)
		}

		sortVal := math.NaN()
		if sv := get(si); sv != "" {
			if n, err := strconv.ParseFloat(sv, 64); err == nil {
				sortVal = n
			}
		}

		sectionItems[id] = append(sectionItems[id], rawItem{
			name:      itemName,
			price:     get(pi),
			sortOrder: sortVal,
			rowNum:    rowNum,
		})
	}

	// Build sections, applying sort order
	sections := make([]Section, 0, len(sectionOrder))
	for _, id := range sectionOrder {
		raw := sectionItems[id]

		// Split into explicitly ordered vs naturally ordered
		var explicit, natural []rawItem
		for _, it := range raw {
			if math.IsNaN(it.sortOrder) {
				natural = append(natural, it)
			} else {
				explicit = append(explicit, it)
			}
		}

		// Sort the explicit group by their sort order value
		sort.SliceStable(explicit, func(i, j int) bool {
			return explicit[i].sortOrder < explicit[j].sortOrder
		})
		// Natural group already preserves CSV row order

		// Explicit items first, then naturally-ordered items
		all := append(explicit, natural...)

		items := make([]Item, len(all))
		for i, it := range all {
			items[i] = Item{Name: it.name, Price: it.price}
		}

		sections = append(sections, Section{
			ID:    id,
			Name:  sectionNames[id],
			Items: items,
		})
	}

	return sections, nil
}

// ─── HTML template ───────────────────────────────────────────────────────────

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>El Classico – Menu</title>
  <link href="https://fonts.googleapis.com/css2?family=UnifrakturMaguntia&family=Cinzel+Decorative:wght@700&family=Cinzel:wght@400;600&family=IM+Fell+English:ital@0;1&display=swap" rel="stylesheet" />
  <style>
    :root {
      --parchment:      #c8a96e;
      --parchment-dark: #b08d50;
      --ink:            #1a0e00;
      --ink-mid:        #3b2200;
      --rule:           #6b4218;
    }
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #6b4a1e;
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 2rem 1rem 4rem;
      min-height: 100vh;
      font-family: 'IM Fell English', serif;
    }
    .page {
      background: var(--parchment);
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='400' height='400'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.75' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='400' height='400' filter='url(%23n)' opacity='0.08'/%3E%3C/svg%3E");
      width: min(720px, 100%);
      padding: 2.5rem 2.8rem 3rem;
      margin-bottom: 2.5rem;
      position: relative;
      box-shadow: 0 8px 40px #0008, inset 0 0 80px #0002;
    }
    .page::before, .page::after,
    .page .corner-bl::before, .page .corner-br::before {
      content: '❧';
      position: absolute;
      font-size: 2.4rem;
      color: var(--rule);
      line-height: 1;
    }
    .page::before  { top: .6rem;  left: .8rem;  transform: scaleX(-1); }
    .page::after   { top: .6rem;  right: .8rem; }
    .page .corner-bl::before { bottom: .6rem; left: .8rem;  transform: scaleY(-1) scaleX(-1); }
    .page .corner-br::before { bottom: .6rem; right: .8rem; transform: scaleY(-1); }
    .logo-wrap { text-align: center; margin-bottom: 1.8rem; }
    .logo-top-ornament { font-size: 2rem; color: var(--rule); letter-spacing: .3em; display: block; margin-bottom: .3rem; }
    .logo-rule { display: flex; align-items: center; gap: .6rem; justify-content: center; margin-bottom: .5rem; }
    .logo-rule span { flex: 1; height: 2px; background: var(--rule); }
    .logo-rule .dot { width: 6px; height: 6px; border-radius: 50%; background: var(--rule); flex: 0 0 auto; }
    .logo-title {
      font-family: 'Cinzel Decorative', serif;
      font-size: clamp(2.2rem, 8vw, 3.8rem);
      font-weight: 700;
      color: var(--ink);
      text-shadow: 3px 3px 0 var(--parchment-dark), 6px 6px 0 #0002;
      letter-spacing: .12em;
      line-height: 1;
    }
    .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 2rem 2.4rem; margin-top: 1.2rem; }
    @media (max-width: 540px) { .grid { grid-template-columns: 1fr; } }
    .section-heading {
      font-family: 'UnifrakturMaguntia', cursive;
      font-size: 1.55rem;
      color: var(--ink);
      letter-spacing: .04em;
      margin-bottom: .35rem;
      line-height: 1.1;
    }
    .section-rule { height: 2px; background: linear-gradient(to right, var(--rule), transparent); margin-bottom: .75rem; border: none; }
    .item { display: flex; justify-content: space-between; align-items: baseline; gap: .5rem; margin-bottom: .28rem; font-size: .97rem; color: var(--ink-mid); line-height: 1.35; }
    .item-name { flex: 1; }
    .item-price { font-family: 'Cinzel', serif; font-size: .85rem; color: var(--ink); white-space: nowrap; font-weight: 600; }
    .note { font-style: italic; font-size: .88rem; color: var(--ink-mid); margin-top: .8rem; text-align: center; }
    .generated { text-align: center; font-size: .78rem; color: #fff8; margin-top: 1rem; font-family: monospace; }
  </style>
</head>
<body>

<div class="page">
  <div class="corner-bl"></div>
  <div class="corner-br"></div>
  <div class="logo-wrap">
    <span class="logo-top-ornament">✦ ✦ ✦</span>
    <div class="logo-rule"><span></span><span class="dot"></span><span></span></div>
    <div class="logo-title">El Classico</div>
    <div class="logo-rule" style="margin-top:.5rem"><span></span><span class="dot"></span><span></span></div>
  </div>
  <div class="grid">
    {{range .Sections}}
    <div class="section" id="{{.ID}}">
      <div class="section-heading">{{.Name}}</div>
      <hr class="section-rule" />
      {{range .Items}}
      <div class="item">
        <span class="item-name">{{.Name}}</span>
        {{if .Price}}<span class="item-price">{{.Price}}</span>{{end}}
      </div>
      {{end}}
    </div>
    {{end}}
  </div>
  <p class="note">Note: we have some items in addition to this menu. Please ask the chef!</p>
</div>

<p class="generated">Generated from {{.CSVFile}}</p>
</body>
</html>`

type templateData struct {
	Sections []Section
	CSVFile  string
}

// ─── main ────────────────────────────────────────────────────────────────────

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}

func defaultPath(override, filename string) string {
	if override != "" {
		return override
	}
	exe, err := os.Executable()
	if err != nil {
		return filename
	}
	return filepath.Join(filepath.Dir(exe), filename)
}

func main() {
	args := os.Args[1:]

	csvPath := ""
	outPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--csv":
			if i+1 >= len(args) {
				fatal("--csv requires a value")
			}
			csvPath = args[i+1]
			i++
		case "--out":
			if i+1 >= len(args) {
				fatal("--out requires a value")
			}
			outPath = args[i+1]
			i++
		case "-h", "--help", "help":
			fmt.Print(`El Classico — menu HTML generator

USAGE
  elclassico [--csv menu.csv] [--out index.html]

FLAGS
  --csv <path>   Input CSV file   (default: menu.csv next to binary)
  --out <path>   Output HTML file (default: index.html next to binary)

CSV FORMAT
  Required columns (header row, any order):
    Section    — section display name  e.g. "Biryani"
    Menu item  — item name             e.g. "Chicken Dum Biryani"
    Price      — price string          e.g. "129/229" or "149"
    Sort order — optional number; items without a sort order appear
                 after sorted items, in their original CSV row order

EXAMPLES
  elclassico
  elclassico --csv data/menu.csv
  elclassico --csv menu.csv --out public/index.html
`)
			os.Exit(0)
		default:
			fatal(fmt.Sprintf("unknown argument %q — run 'elclassico --help'", args[i]))
		}
	}

	csvPath = defaultPath(csvPath, "menu.csv")
	outPath = defaultPath(outPath, "index.html")

	sections, err := loadCSV(csvPath)
	if err != nil {
		fatal(err.Error())
	}
	if len(sections) == 0 {
		fatal("no sections found in " + csvPath)
	}

	t, err := template.New("menu").Parse(htmlTemplate)
	if err != nil {
		fatal("template parse error: " + err.Error())
	}

	out, err := os.Create(outPath)
	if err != nil {
		fatal("cannot create " + outPath + ": " + err.Error())
	}
	defer out.Close()

	if err := t.Execute(out, templateData{Sections: sections, CSVFile: filepath.Base(csvPath)}); err != nil {
		fatal("template error: " + err.Error())
	}

	totalItems := 0
	for _, s := range sections {
		totalItems += len(s.Items)
	}

	const green = "\033[32m"
	const reset = "\033[0m"
	fmt.Printf("%sGenerated%s %s  (%d sections, %d items)\n",
		green, reset, outPath, len(sections), totalItems)
}
