# El Classico — Menu Website & CLI

## How it works

You maintain `menu.csv` — a plain spreadsheet anyone can edit.
Run `elclassico` and it generates a fully-styled `index.html` ready for GitHub Pages.

```
menu.csv  →  elclassico  →  index.html
```

---

## CSV format

| Column | Required | Notes |
|---|---|---|
| `Section` | ✓ | Section display name, e.g. `Biryani` |
| `Menu item` | ✓ | Item name, e.g. `Chicken Dum Biryani` |
| `Price` | ✓ | Price string, e.g. `129` or `129/229` for half/full |
| `Sort order` | optional | Number. Items without a sort order appear after sorted items, in CSV row order |

- Column order in the CSV doesn't matter — headers are matched by name.
- Rows with an empty Section **and** empty Menu item are ignored (useful for blank rows in spreadsheets).
- Multiple sections supported — each unique Section value becomes its own block on the page.

---

## Build the CLI

Requires [Go 1.18+](https://go.dev/dl/).

```bash
go mod init elclassico
go build -o elclassico .          # Linux / macOS
go build -o elclassico.exe .      # Windows
```

Place the binary in the same directory as `menu.csv`.

---

## Generate

```bash
elclassico                                         # reads menu.csv, writes index.html
elclassico --csv data/menu.csv                     # custom CSV path
elclassico --csv menu.csv --out public/index.html  # custom output path
elclassico --help                                  # show usage
```

---

## Deploy to GitHub Pages

```bash
git init
git add index.html menu.csv main.go README.md
git commit -m "Initial El Classico menu"
git branch -M main
git remote add origin https://github.com/<you>/<repo>.git
git push -u origin main
```

Then: **Settings → Pages → Source: Deploy from branch → main / (root)**

Your menu will be live at `https://<you>.github.io/<repo>/`

---

## Workflow: edit → regenerate → deploy

1. Edit `menu.csv` (Excel, Google Sheets, or any text editor).
2. Regenerate:
   ```bash
   ./elclassico
   ```
3. Commit and push:
   ```bash
   git add menu.csv index.html && git commit -m "Update menu" && git push
   ```
4. GitHub Pages redeploys in ~30 seconds.