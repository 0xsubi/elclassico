// elclassico — CLI tool to manage the El Classico restaurant menu.
//
// Data is stored in menu.json (same directory as the binary by default,
// override with --data flag).
//
// Usage:
//   elclassico list
//   elclassico list <section-id>
//   elclassico item add    <section-id> "<name>" <price>
//   elclassico item edit   <section-id> <item-index> [--name "<new>"] [--price <new>]
//   elclassico item delete <section-id> <item-index>
//   elclassico section add    <id> "<name>" [--page <n>]
//   elclassico section rename <section-id> "<new-name>"
//   elclassico section delete <section-id>
//   elclassico generate [--out index.html]

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ─── data model ──────────────────────────────────────────────────────────────

type Item struct {
	Name  string `json:"name"`
	Price string `json:"price"`
}

type Section struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Page  int    `json:"page"`
	Items []Item `json:"items"`
}

type Menu struct {
	Sections []Section `json:"sections"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func dataPath(override string) string {
	if override != "" {
		return override
	}
	exe, err := os.Executable()
	if err != nil {
		return "menu.json"
	}
	return filepath.Join(filepath.Dir(exe), "menu.json")
}

func load(path string) (*Menu, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var m Menu
	if err := json.Unmarshal(f, &m); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}
	return &m, nil
}

func save(path string, m *Menu) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func findSection(m *Menu, id string) (int, error) {
	for i, s := range m.Sections {
		if strings.EqualFold(s.ID, id) {
			return i, nil
		}
	}
	return -1, fmt.Errorf("section %q not found", id)
}

func parseIndex(s string, max int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, errors.New("item index must be a number")
	}
	if n < 1 || n > max {
		return 0, fmt.Errorf("index %d out of range (1–%d)", n, max)
	}
	return n - 1, nil // convert to 0-based
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// ─── display ─────────────────────────────────────────────────────────────────

const (
	bold  = "\033[1m"
	cyan  = "\033[36m"
	green = "\033[32m"
	reset = "\033[0m"
	dim   = "\033[2m"
)

func printSection(idx int, s Section) {
	fmt.Printf("\n%s[%s]%s %s%s%s %s(page %d)%s\n",
		cyan, s.ID, reset,
		bold, s.Name, reset,
		dim, s.Page, reset)
	fmt.Println(strings.Repeat("─", 48))
	for i, it := range s.Items {
		fmt.Printf("  %s%2d.%s  %-36s %s₹ %s%s\n",
			dim, i+1, reset, it.Name, green, it.Price, reset)
	}
}

// ─── commands ────────────────────────────────────────────────────────────────

func cmdList(m *Menu, args []string) {
	if len(args) == 0 {
		fmt.Printf("\n%sEl Classico — Menu%s\n", bold, reset)
		fmt.Println(strings.Repeat("═", 50))
		for _, s := range m.Sections {
			printSection(0, s)
		}
		return
	}
	idx, err := findSection(m, args[0])
	if err != nil {
		fatal(err)
	}
	printSection(idx, m.Sections[idx])
}

func cmdItemAdd(m *Menu, args []string) error {
	// item add <section-id> "<name>" <price>
	if len(args) < 3 {
		return errors.New("usage: item add <section-id> \"<name>\" <price>")
	}
	si, err := findSection(m, args[0])
	if err != nil {
		return err
	}
	name := args[1]
	price := args[2]
	m.Sections[si].Items = append(m.Sections[si].Items, Item{Name: name, Price: price})
	fmt.Printf("%sAdded%s %q to [%s] at ₹%s\n", green, reset, name, m.Sections[si].ID, price)
	return nil
}

func cmdItemEdit(m *Menu, args []string, flags map[string]string) error {
	// item edit <section-id> <index> [--name "..."] [--price ...]
	if len(args) < 2 {
		return errors.New("usage: item edit <section-id> <index> [--name <new>] [--price <new>]")
	}
	si, err := findSection(m, args[0])
	if err != nil {
		return err
	}
	ii, err := parseIndex(args[1], len(m.Sections[si].Items))
	if err != nil {
		return err
	}
	item := &m.Sections[si].Items[ii]
	if n, ok := flags["name"]; ok && n != "" {
		item.Name = n
	}
	if p, ok := flags["price"]; ok && p != "" {
		item.Price = p
	}
	fmt.Printf("%sUpdated%s item %d in [%s]: %q @ ₹%s\n",
		green, reset, ii+1, m.Sections[si].ID, item.Name, item.Price)
	return nil
}

func cmdItemDelete(m *Menu, args []string) error {
	// item delete <section-id> <index>
	if len(args) < 2 {
		return errors.New("usage: item delete <section-id> <index>")
	}
	si, err := findSection(m, args[0])
	if err != nil {
		return err
	}
	ii, err := parseIndex(args[1], len(m.Sections[si].Items))
	if err != nil {
		return err
	}
	removed := m.Sections[si].Items[ii]
	m.Sections[si].Items = append(m.Sections[si].Items[:ii], m.Sections[si].Items[ii+1:]...)
	fmt.Printf("%sDeleted%s %q from [%s]\n", green, reset, removed.Name, m.Sections[si].ID)
	return nil
}

func cmdSectionAdd(m *Menu, args []string, flags map[string]string) error {
	// section add <id> "<name>" [--page n]
	if len(args) < 2 {
		return errors.New("usage: section add <id> \"<name>\" [--page <n>]")
	}
	id := slugify(args[0])
	name := args[1]
	page := 1
	if p, ok := flags["page"]; ok {
		n, err := strconv.Atoi(p)
		if err != nil {
			return errors.New("--page must be a number")
		}
		page = n
	}
	// check duplicate
	if _, err := findSection(m, id); err == nil {
		return fmt.Errorf("section %q already exists", id)
	}
	m.Sections = append(m.Sections, Section{ID: id, Name: name, Page: page})
	fmt.Printf("%sAdded section%s [%s] %q on page %d\n", green, reset, id, name, page)
	return nil
}

func cmdSectionRename(m *Menu, args []string) error {
	// section rename <id> "<new-name>"
	if len(args) < 2 {
		return errors.New("usage: section rename <section-id> \"<new-name>\"")
	}
	si, err := findSection(m, args[0])
	if err != nil {
		return err
	}
	old := m.Sections[si].Name
	m.Sections[si].Name = args[1]
	fmt.Printf("%sRenamed%s section [%s] from %q to %q\n", green, reset, m.Sections[si].ID, old, args[1])
	return nil
}

func cmdSectionDelete(m *Menu, args []string) error {
	// section delete <id>
	if len(args) < 1 {
		return errors.New("usage: section delete <section-id>")
	}
	si, err := findSection(m, args[0])
	if err != nil {
		return err
	}
	name := m.Sections[si].Name
	m.Sections = append(m.Sections[:si], m.Sections[si+1:]...)
	fmt.Printf("%sDeleted%s section %q\n", green, reset, name)
	return nil
}

// ─── flag parsing ─────────────────────────────────────────────────────────────

// parseFlags splits args into positional args and --key value flags.
func parseFlags(args []string) ([]string, map[string]string) {
	positional := []string{}
	flags := map[string]string{}
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") {
			key := strings.TrimPrefix(args[i], "--")
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				flags[key] = args[i+1]
				i++
			} else {
				flags[key] = "true"
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return positional, flags
}

// ─── generate ────────────────────────────────────────────────────────────────

// htmlEscape makes content safe to embed in HTML text nodes.
// html/template handles this automatically, but we keep the import tidy.
var _ = template.HTMLEscapeString // ensure import is used

// groupByPage splits sections into a map keyed by page number, preserving order.
func groupByPage(m *Menu) [][]Section {
	pageMap := map[int][]Section{}
	pageOrder := []int{}
	for _, s := range m.Sections {
		if _, seen := pageMap[s.Page]; !seen {
			pageOrder = append(pageOrder, s.Page)
		}
		pageMap[s.Page] = append(pageMap[s.Page], s)
	}
	result := make([][]Section, len(pageOrder))
	for i, p := range pageOrder {
		result[i] = pageMap[p]
	}
	return result
}

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
    .callout { border: 2px solid var(--rule); border-radius: 2px; padding: .8rem 1.2rem; margin-top: 1rem; text-align: center; }
    .callout p { font-style: italic; font-size: .95rem; color: var(--ink-mid); line-height: 1.6; }
    .callout a { color: var(--ink); }
    .note { font-style: italic; font-size: .88rem; color: var(--ink-mid); margin-top: .8rem; text-align: center; }
    .generated { text-align: center; font-size: .78rem; color: #fff8; margin-top: 1rem; font-family: monospace; }
  </style>
</head>
<body>
{{range .Pages}}
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
        <span class="item-price">{{.Price}}</span>
      </div>
      {{end}}
    </div>
    {{end}}
  </div>
  {{if .IsLast}}
  <p class="note">Note: we have some items in addition to this menu. Please ask the chef!</p>
  {{end}}
</div>
{{end}}
<p class="generated">Generated by elclassico CLI · {{.Generated}}</p>
</body>
</html>`

type pageData struct {
	Sections []Section
	IsLast   bool
}

type templateData struct {
	Pages     []pageData
	Generated string
}

func cmdGenerate(m *Menu, flags map[string]string) error {
	outPath := "index.html"
	if p, ok := flags["out"]; ok && p != "" {
		outPath = p
	}

	pages := groupByPage(m)
	pd := make([]pageData, len(pages))
	for i, secs := range pages {
		pd[i] = pageData{Sections: secs, IsLast: i == len(pages)-1}
	}

	t, err := template.New("menu").Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("cannot create %s: %w", outPath, err)
	}
	defer f.Close()

	data := templateData{
		Pages:     pd,
		Generated: "menu.json",
	}
	if err := t.Execute(f, data); err != nil {
		return fmt.Errorf("template execute error: %w", err)
	}

	fmt.Printf("%sGenerated%s %s  (%d sections across %d page(s))\n",
		green, reset, outPath, len(m.Sections), len(pages))
	return nil
}

// ─── help ────────────────────────────────────────────────────────────────────

func printHelp() {
	fmt.Print(`
El Classico Menu CLI

USAGE
  elclassico [--data <path>] <command> [args] [flags]

COMMANDS
  list [<section-id>]
      List all sections (or a single section) with items.

  item add <section-id> "<name>" <price>
      Add a new item to a section.

  item edit <section-id> <index> [--name "<new>"] [--price <new>]
      Edit an existing item (1-based index).

  item delete <section-id> <index>
      Remove an item (1-based index).

  section add <id> "<name>" [--page <n>]
      Add a new section (id auto-slugified).

  section rename <section-id> "<new-name>"
      Rename a section.

  section delete <section-id>
      Delete a section and all its items.

  generate [--out <path>]
      Regenerate index.html from menu.json. Default output: index.html

GLOBAL FLAGS
  --data <path>   Path to menu.json  (default: menu.json next to binary)

EXAMPLES
  elclassico list
  elclassico list biryani
  elclassico item add biryani "Prawn Biryani" 349
  elclassico item edit chinese 3 --name "Veg Manchurian" --price 149
  elclassico item delete beverages 2
  elclassico section add desserts "Desserts" --page 2
  elclassico section rename desserts "Sweet Corner"
  elclassico section delete desserts
  elclassico generate
  elclassico generate --out public/index.html
`)
}

// ─── main ────────────────────────────────────────────────────────────────────

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printHelp()
		os.Exit(0)
	}

	// extract global --data flag first
	dataOverride := ""
	filtered := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--data" && i+1 < len(args) {
			dataOverride = args[i+1]
			i++
		} else {
			filtered = append(filtered, args[i])
		}
	}
	args = filtered

	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		printHelp()
		os.Exit(0)
	}

	path := dataPath(dataOverride)
	m, err := load(path)
	if err != nil {
		fatal(err)
	}

	cmd := args[0]
	rest := args[1:]
	positional, flags := parseFlags(rest)

	var cmdErr error

	switch cmd {
	case "list":
		cmdList(m, positional)
		return // no save needed

	case "generate":
		if err := cmdGenerate(m, flags); err != nil {
			fatal(err)
		}
		return // no save needed — generate is read-only

	case "item":
		if len(positional) == 0 {
			fatal(errors.New("usage: item <add|edit|delete> ..."))
		}
		sub := positional[0]
		subArgs := positional[1:]
		switch sub {
		case "add":
			cmdErr = cmdItemAdd(m, subArgs)
		case "edit":
			cmdErr = cmdItemEdit(m, subArgs, flags)
		case "delete":
			cmdErr = cmdItemDelete(m, subArgs)
		default:
			fatal(fmt.Errorf("unknown item subcommand %q", sub))
		}

	case "section":
		if len(positional) == 0 {
			fatal(errors.New("usage: section <add|rename|delete> ..."))
		}
		sub := positional[0]
		subArgs := positional[1:]
		switch sub {
		case "add":
			cmdErr = cmdSectionAdd(m, subArgs, flags)
		case "rename":
			cmdErr = cmdSectionRename(m, subArgs)
		case "delete":
			cmdErr = cmdSectionDelete(m, subArgs)
		default:
			fatal(fmt.Errorf("unknown section subcommand %q", sub))
		}

	default:
		fatal(fmt.Errorf("unknown command %q — run 'elclassico help'", cmd))
	}

	if cmdErr != nil {
		fatal(cmdErr)
	}

	if err := save(path, m); err != nil {
		fatal(fmt.Errorf("could not save menu: %w", err))
	}
}
