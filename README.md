# El Classico — Menu Website & CLI

## Website (GitHub Pages)

`index.html` is a self-contained static page — no build step needed.

### Deploy to GitHub Pages

```bash
git init
git add index.html menu.json main.go README.md
git commit -m "Initial El Classico menu"
git branch -M main
git remote add origin git@github.com:0xsubi/elclassico.git
git push -u origin main
```

Then in your repo → **Settings → Pages → Source: Deploy from branch → main / (root)**.

Your menu will be live at `https://<your-username>.github.io/<your-repo>/`.

---

## CLI — Build

Requires [Go 1.18+](https://go.dev/dl/).

```bash
# one-time setup
go mod init elclassico

# build
go build -o elclassico .          # Linux / macOS
go build -o elclassico.exe .      # Windows
```

Place the `elclassico` binary and `menu.json` in the **same directory**.

---

## CLI — Usage

```
elclassico [--data <path>] <command> [args] [flags]
```

### List

```bash
elclassico list                   # all sections
elclassico list biryani           # one section by ID
```

### Generate HTML from menu.json

After any edit, regenerate `index.html` in one command:

```bash
elclassico generate                          # writes index.html (default)
elclassico generate --out public/index.html  # custom output path
```

This rebuilds the full parchment-styled webpage from the current `menu.json` — every section, every item, every price — so the site always stays in sync with your data.

### Items

```bash
# Add an item
elclassico item add <section-id> "<name>" <price>
elclassico item add biryani "Prawn Biryani" 349
elclassico item add biryani "Half/Full Chicken Biryani" "149/299"

# Edit an item  (1-based index shown in `list`)
elclassico item edit <section-id> <index> [--name "<new>"] [--price <new>]
elclassico item edit chinese 3 --price 149
elclassico item edit beverages 1 --name "Red Tea" --price 15

# Delete an item
elclassico item delete <section-id> <index>
elclassico item delete specials 2
```

### Sections

```bash
# Add a section (ID is auto-slugified from the first arg)
elclassico section add <id> "<display name>" [--page <n>]
elclassico section add desserts "Desserts" --page 2

# Rename display name (ID stays the same)
elclassico section rename <section-id> "<new name>"
elclassico section rename desserts "Sweet Corner"

# Delete a section (removes all its items too)
elclassico section delete <section-id>
elclassico section delete desserts
```

### Override data file path

```bash
elclassico --data /path/to/menu.json list
```

---

## Workflow: edit menu → redeploy

1. Edit with the CLI:
   ```bash
   ./elclassico item add biryani "Fish Biryani" 249
   ```
2. Regenerate the HTML:
   ```bash
   ./elclassico generate
   ```
3. Commit both updated files:
   ```bash
   git add menu.json index.html && git commit -m "Add Fish Biryani"
   git push
   ```
4. GitHub Pages auto-redeploys in ~30 seconds.