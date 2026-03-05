# OPS Services Reference

Base URL: `https://ops.epo.org/rest-services/`
Auth header: `Authorization: Bearer <token>`

For JSON response: `Accept: application/json` (works on all XML-supporting services)
Quick JSON shortcut: append `.json` to any URI.

---

## 1. Published-Data Service

### Generic Request Structure

```
GET /published-data/{reference-type}/{input-format}/{input}/[endpoint]/[constituents]
```

- **reference-type**: `publication` | `application` | `priority`
- **input-format**: `docdb` | `epodoc`
- **input**: e.g. `EP1000000`, `EP1000000.A1`
- **endpoint** (optional): `biblio`, `abstract`, `fulltext`, `claims`, `description`, `images`, `equivalents`
- **constituents** (optional): `biblio`, `abstract`, `full-cycle`, `images` (comma-separated)

Omitting endpoint defaults to `biblio`.

---

### Bibliographic Data

```bash
# Single patent
GET /published-data/publication/epodoc/EP1000000.A1/biblio
Accept: application/exchange+xml

# Without kind code → returns all publications for this application
GET /published-data/publication/epodoc/EP1000000/biblio

# With full-cycle (all publications in lifecycle)
GET /published-data/publication/epodoc/EP1000000.A1/biblio,full-cycle

# Abstract only
GET /published-data/publication/epodoc/EP1000000.A1/abstract

# Bulk GET (small sets)
GET /published-data/publication/epodoc/EP1676595,JP2006187606,US2006142694/biblio

# Bulk POST (up to 100 references)
POST /published-data/publication/epodoc/biblio
Content-Type: text/plain

EP1676595
JP2006187606
US2006142694
```

---

### Fulltext

Two-step: inquire first, then retrieve.

```bash
# Step 1: Check availability
GET /published-data/publication/epodoc/EP1000000/fulltext
Accept: application/fulltext+xml

# Step 2a: Get description
GET /published-data/publication/epodoc/EP1000000/description
Accept: application/fulltext+xml

# Step 2b: Get claims
GET /published-data/publication/epodoc/EP1000000/claims
Accept: application/fulltext+xml
```

**Supported authorities for fulltext:**
EP, WO, AT, BE, BG, CA, CH, CY, CZ, DK, EE, ES, FR, GB, GR, HR, IE, IT, LT, LU, MC, MD, ME, NO, PL, PT, RO, RS, SE, SK

---

### Images

Two-step: inquire to get links, then retrieve.

```bash
# Step 1: Get available image links
GET /published-data/publication/epodoc/EP1000000.A1/images
Accept: application/ops+xml

# Step 2: Retrieve by link from Step 1 response
GET /published-data/images/EP/1000000/PA/firstpage
Accept: image/png        # or image/pdf, application/tiff

# All thumbnails (PDF)
GET /published-data/images/EP/1000000/A1/thumbnail
Accept: application/pdf

# Single page from full document
GET /published-data/images/EP/1000000/A1/fullimage
Accept: application/pdf
X-OPS-Range: 1

# TIFF thumbnail (requires range)
GET /published-data/images/EP/1000000/A1/thumbnail.tiff?Range=4
```

**Image formats:** `image/pdf`, `application/tiff`, `image/png`, `image/jpeg`

`firstpage.jpeg` returns max 320px wide. PNG/TIFF are original size.

---

### Equivalents (Simple Family)

```bash
GET /published-data/publication/epodoc/EP1000000/equivalents
GET /published-data/publication/epodoc/EP1000000/equivalents/abstract
GET /published-data/publication/epodoc/EP1000000.A1/equivalents/biblio
GET /published-data/publication/epodoc/EP1000000.A1/equivalents/biblio,full-cycle
GET /published-data/publication/epodoc/EP1000000.A1/equivalents/images
```

---

### Bibliographic Search

```bash
GET /published-data/search?q=applicant%3DIBM
GET /published-data/search/biblio,abstract,full-cycle?q=applicant%3DIBM

# POST
POST /published-data/search
Content-Type: text/plain

q=applicant%3DIBM

# Pagination (default 1-25, max 100, total max 2000)
X-OPS-Range: 26-50
```

---

## 2. Family Service

INPADOC extended patent family — relatives sharing at least one priority.

```bash
# Family members only (no biblio)
GET /family/publication/docdb/EP.1000000.A1

# With bibliographic data
GET /family/publication/docdb/EP.1000000.A1/biblio

# With legal status
GET /family/publication/docdb/EP.1000000.A1/legal

# With both
GET /family/publication/docdb/EP.1000000.A1/biblio,legal

# By application number
GET /family/application/epodoc/EP99203729

# By priority
GET /family/priority/docdb/US.18314305.A

# Wildcard kind code
GET /family/publication/docdb/EP.1000000.*
GET /family/publication/docdb/EP.1000000.A*

# POST
POST /family/publication/docdb
Content-Type: text/plain

EP.1000000.A1
```

**Note:** If family is very large (100s of members), response will be truncated with `truncatedFamily="true"`.

---

## 3. Number Service

Convert between `original`, `docdb`, `epodoc` formats.

```bash
# docdb → epodoc
GET /number-service/application/docdb/MD.20050130.A.20050130/epodoc

# original → docdb (Japan)
GET /number-service/application/original/JP.(2006-147056).A.20060526/docdb

# docdb → original (Japan)
GET /number-service/application/docdb/JP.2006147056.A.20060526/original

# original → epodoc (USPTO)
GET /number-service/application/original/US.(08/921,321).A.19970829/epodoc

# PCT original → docdb
GET /number-service/application/original/(PCT/GB02/04635).20021011/docdb

# POST
POST /number-service/application/docdb/epodoc
Content-Type: text/plain

MD.20050130.A.20050130
```

**Conversion matrix:**
```
original  → docdb, epodoc
docdb     → epodoc, original
epodoc    → original
```

---

## 4. Register Service

European Patent Register data. EP-only. Input format: `epodoc` only.

```bash
# Bibliographic data (default)
GET /register/application/epodoc/EP99203729
Accept: application/register+xml

# Events (dossier actions)
GET /register/application/epodoc/EP99203729/events

# Procedural steps
GET /register/application/epodoc/EP99203729/procedural-steps

# Unitary Patent Protection
GET /register/publication/epodoc/EP99203729/upp

# Combined
GET /register/application/epodoc/EP99203729/biblio,events,procedural-steps

# Search
GET /register/search/?q=pa%3DIBM
Accept: application/register+xml

# POST search
POST /register/search
Content-Type: text/plain

q=pa%3DIBM
```

Using combined constituents is generally more quota-efficient than separate calls because OPS returns dossier slices in one request envelope.

**Pagination:** default 1–25, max 100 via `Range` header.

---

## 5. Legal Service

Legal event data for the full patent lifecycle (INPADOC).

```bash
GET /legal/publication/docdb/EP.1000000.A1
Accept: application/ops+xml

POST /legal/publication/docdb
Content-Type: text/plain

EP.1000000.A1
```

Response includes `ops:legal` elements per family member with:
- `code` — legal event code (e.g., `AK`, `RER`)
- `desc` — human-readable description
- `L001EP–L500EP` — structured legal data fields

---

## 6. Classification (CPC)

```bash
# Retrieve a CPC symbol
GET /classification/cpc/A
GET /classification/cpc/A62C37/48

# With children (depth)
GET /classification/cpc/A?depth=1
GET /classification/cpc/H04W?depth=all   # only for level > 5

# With navigation (prev/next)
GET /classification/cpc/A01?navigation

# With ancestors
GET /classification/cpc/A01?navigation&ancestors

# Search by keyword
GET /classification/cpc/search/?q=chemistry
GET /classification/cpc/search/?q=chemistry&Range=1-20

# Retrieve media from classification text
GET /classification/cpc/media/1000.gif
Accept: image/gif

# Classification mapping (ECLA ↔ CPC ↔ IPC)
GET /classification/map/ecla/A61K9/00/cpc
GET /classification/map/cpc/A01D2085/008/ecla?additional

# POST
POST /classification/cpc
Content-Type: text/plain

A01B
```

**Mapping input → output formats:**
```
ecla  → cpc, ipc
cpc   → ecla, ipc
```

---

## 7. Data Usage API

```bash
GET https://ops.epo.org/3.2/developers/me/stats/usage?timeRange=30/03/2024
GET https://ops.epo.org/3.2/developers/me/stats/usage?timeRange=01/03/2024~31/03/2024
Authorization: Bearer <token>
```

---

## Content Types Reference

| Service | Accept Header |
|---------|--------------|
| Published-data (XML) | `application/exchange+xml` |
| Published-data (fulltext) | `application/fulltext+xml` |
| Images | `image/pdf`, `application/tiff`, `image/png` |
| Number-service | `application/ops+xml` |
| Family | `application/ops+xml` |
| Legal | `application/ops+xml` |
| Classification/CPC | `application/cpc+xml` |
| Register | `application/register+xml` |
| **Any XML service** | `application/json` or `application/javascript` |

**JSON shortcut:** Add `.json` to any OPS URI.
```
GET /published-data/publication/epodoc/EP1000000/biblio.json
```

**JSON format:** OPS uses BadgerFish convention (XML attributes → `@`, text content → `$`).
