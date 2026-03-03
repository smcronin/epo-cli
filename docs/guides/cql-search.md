# CQL Search Syntax

OPS uses Common Query Language (CQL) for bibliographic search (`/published-data/search`).

> **Note:** CQL for Register search (`/register/search`) uses different identifiers. This page covers Published-data only.

---

## Basic Rules

1. A query is one or more search clauses connected by Boolean operators
2. A search clause: `index=term` (or just `term` if index is auto-detected)
3. Missing index → OPS auto-detects based on term shape:
   - 2-letter ISO code → `num`
   - Date pattern → `pd`
   - CPC-like pattern (e.g. `H04W`) → `cl`
   - Alphanumeric + digits → `num`
   - All letters, capitalized → `ia`
   - Otherwise → `txt`

---

## Index Reference

| Index | Alias | Description |
|-------|-------|-------------|
| `title` | `ti` | Title (English) |
| `abstract` | `ab` | Abstract (English) |
| `titleandabstract` | `ta` | Title OR abstract |
| `inventor` | `in` | Inventor name |
| `applicant` | `pa` | Applicant name |
| `inventorandapplicant` | `ia` | Either name field |
| `publicationnumber` | `pn` | Publication number (any format) |
| `spn` | | Publication number (epodoc) |
| `applicantnumber` | `ap` | Application number (any format) |
| `sap` | | Application number (epodoc) |
| `prioritynumber` | `pr` | Priority number |
| `spr` | | Priority number (epodoc) |
| `num` | | Publication, application, or priority (any format) |
| `publicationdate` | `pd` | Publication date |
| `citation` | `ct` | Cited document number |
| `ex` | | Cited during examination |
| `op` | | Cited during opposition |
| `rf` | | Cited by applicant |
| `oc` | | Other citation |
| `famn` | | Simple family identifier |
| `cpc` | | CPC classification |
| `cpci` | | CPC invention classifications |
| `cpca` | | CPC additional classifications |
| `cpcc` | | CPC confirmed + national office |
| `ipc`, `ic` | | IPC1–8 class (any) |
| `ci` | | IPC8 core invention class |
| `cn` | | IPC8 core additional class |
| `ai` | | IPC8 advanced invention class |
| `an` | | IPC8 advanced additional class |
| `cl` | | CPC or IPC class |
| `txt` | | Title, abstract, or names |

---

## Date Formats for `pd`

```
yyyy            # 2024
yyyyMM          # 202403
yyyyMMdd        # 20240315
yyyy-MM         # 2024-03
yyyy-MM-dd      # 2024-03-15
MM/yyyy         # 03/2024
dd/MM/yyyy      # 15/03/2024
MM.yyyy         # 03.2024
dd.MM.yyyy      # 15.03.2024
```

---

## Boolean Operators

| Operator | Meaning |
|----------|---------|
| `and` | Both must match |
| `or` | Either must match |
| `not` | Must NOT match second clause (`and not`) |

Note: `not` cannot start a query: ~~`not pd=2010`~~ is illegal.

---

## Relation Operators

| Operator | Use |
|----------|-----|
| `=` | Equality (exact match for most fields) |
| `<`, `>`, `<=`, `>=` | Date/numeric ranges |
| `within` | Range: `pd within "20051212 20051214"` |
| `any` | Any of listed words |
| `all` | All listed words |

---

## Wildcards

| Symbol | Meaning |
|--------|---------|
| `*` | Any string (unlimited) |
| `?` | Any single character or none |
| `#` | Any single character (mandatory) |

Examples:
```
ta=synchroni#ed       # synchronized OR synchronised
pa=IBM*               # IBM, IBMS, IBM Corp, ...
pn=EP10000*           # any EP pub starting with 10000
```

Note: prefix wildcards only work in `title` and `abstract` indices.

---

## Proximity Operators

```
ta=green prox/unit=paragraph ta=energy
# Both words in same paragraph

ta=green prox/distance<=3 ta=energy
# Within 3 words of each other (any order)

ta=green prox/distance<=2/ordered=true ta=energy
# "green" followed by "energy" within 2 words
```

---

## CPC Relation Qualifiers

Use with classification indices only:

| Qualifier | Meaning |
|-----------|---------|
| `=/low` | This symbol and all lower hierarchy levels |
| `=/high` | This symbol and all higher hierarchy levels |
| `=/same` | Exact match only |

```
cpc=/low A01B         # A01B and all subclasses
cpc=(C08F prox/unit=sentence (US, EP))   # Classified by both US and EP
```

---

## Query Examples

```
# Exact phrase in title
ti="green energy technology"

# Words in same paragraph
ti=green prox/unit=paragraph ti=energy

# Date range
pd within "20051212 20051214"
pd="20051212 20051214"        # same thing

# Multiple words in applicant name
pa all "intelligence agency atomic"

# EP patents by IBM, published after 2000
pa all "intelligence agency atomic" and JP and pd>2000

# Citations
ct=EP1027777                  # citing EP1027777

# Country + year + name (auto-detected)
EP and 2009 and Smith

# CPC subclass hierarchy
cpc=/low A01B

# Published before 18th century
pd < 18000101

# Masked character (British vs. American spelling)
ta=synchroni#ed

# Title or abstract with proximity
(ta=green prox/distance<=3 ta=energy) or (ta=renewable prox/distance<=3 ta=energy)
```

---

## Extended CQL (Smart Search Shorthand)

OPS supports simplified input that auto-expands:

| Input | Equivalent CQL |
|-------|---------------|
| `G08B25 H04L63` | `cl=G08B25 and cl=H04L63` |
| `G08B25 H04L63 title=grid` | `cl=G08B25 and cl=H04L63 and title=grid` |
| `Siemens EP 200701` | `inventorandapplicant=Siemens and num=EP and publicationdate=200701` |
| `20070101:20070115` | `publicationdate within "20070101 20070115"` |

---

## Range Control

Default: 25 results per page. Maximum: 100 per request. Maximum total: 2000.

```bash
# Via header
GET /published-data/search?q=applicant%3DIBM
X-OPS-Range: 50-75

# Via query param (test only)
GET /published-data/search?q=applicant%3DIBM&Range=50-75
```

Response includes `total-result-count` (capped at 10,000 even if more exist).

---

## URL Encoding Reference

Required special chars:
```
= → %3D
: → %3A
space → %20
, → %2C
```

Example query `applicant=IBM`:
```
?q=applicant%3DIBM
```
