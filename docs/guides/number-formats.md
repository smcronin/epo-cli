# Number Formats

OPS uses three number formats: `original`, `docdb`, and `epodoc`. Most services require `docdb` or `epodoc`.

---

## Format Overview

| Format | Description | When to use |
|--------|-------------|-------------|
| `original` | Domestic format as printed on the document | Input only (number-service) |
| `docdb` | EPO normalized format: `CC.number.KC[.date]` | Most OPS services |
| `epodoc` | EPO strict format: `CCnumberKC` (no dots) | Most OPS services, search |

---

## Format Examples (Application Number)

| Format | Example |
|--------|---------|
| original | `MD a 2005 0130` |
| docdb | `MD.20050130.A` |
| epodoc | `MD20050000130` |

---

## Input Construction Rules

### Rule 1: Dot notation (docdb)
Parts separated by dots: `CC.number.KC.date`

```
US.92132197.A.19970829   # CC.number.KC.date
EP.1000000.A1            # CC.number.KC
```

### Rule 2: Special characters → use brackets

Numbers with `/`, `.`, `,` must be wrapped in parens:

```
(US08/921,321)           # slashes and commas
CH.(99947655.9)          # dot in number
```

Or URL-encode commas: `US08%2C921,321`

### Rule 3: URL encoding (mandatory for OPS)

| Char | Code |
|------|------|
| `?` | `%3F` |
| `@` | `%40` |
| `#` | `%23` |
| `%` | `%25` |
| `,` | `%2C` |
| `:` | `%3A` |
| `=` | `%3D` |
| ` ` | `%20` |

**NEVER encode:** `/` or `\`

### Rule 4: Commas in bulk requests

Commas separate multiple references in bulk GET requests:
```
EP1676595,JP2006187606,US2006142694
```

---

## Kind Codes

Appended to publication numbers to identify publication stage.

Common EP kind codes:

| Code | Meaning |
|------|---------|
| A1 | Published application with search report |
| A2 | Published application without search report |
| A3 | Search report only |
| B1 | Granted patent (new EPC) |
| B2 | Revised patent specification |
| B8 | Corrected patent |
| E | European patent (French territory) |
| T | Translation of EP patent |

Wildcards:
- `A*` matches A1, A2, A3...
- `*` matches any kind code

---

## epodoc Publication Format

```
CCNNNNNNNNNNNN(K)
```

- `CC` = ISO 2-letter country code
- `N...` = up to 12 digits, no spaces
- `K` = optional kind code (1 letter)

Rules for K:
- If kind code starts with `A` → NOT attached (silent)
- If kind code is `D`–`Z` → always attached
- If kind code is `B` or `C` → attached only if needed to differentiate overlapping series

Examples:
```
JP2000177507     # kind code A, not attached
JP3000014B       # kind code B1, B attached
JP3000014U       # kind code U, attached
CN100520025C     # kind code C, attached
DE6610524U       # kind code U, attached
KR200142084Y     # kind code Y1, Y attached
```

---

## PCT Numbers in docdb Format

Format changed Jan 1, 2004:

| Period | Format |
|--------|--------|
| Before 2004 | `CCyynnnnnW` |
| After 2004 | `CCccyynnnnnnW` |

- `CC` = country where filing took place (`IB` = International Bureau)
- `cc` = century (20)
- `yy` = year
- `nnnnnn` = sequential number (6 digits; 5 before 2004)
- `W` = mandatory kind code

Example:
```
PCT/GB02/04635 → GB.0204635.W
```

---

## Number Conversion (Number-service)

Supported conversions:

| From | To |
|------|----|
| original | docdb, epodoc |
| docdb | epodoc, original |
| epodoc | original |

```bash
# original → docdb (Japanese example)
GET https://ops.epo.org/rest-services/number-service/application/original/JP.(2006-147056).A.20060526/docdb

# docdb → epodoc
GET https://ops.epo.org/rest-services/number-service/application/docdb/MD.20050130.A.20050130/epodoc

# original → epodoc (USPTO example)
GET https://ops.epo.org/rest-services/number-service/application/original/US.(08/921,321).A.19970829/epodoc

# PCT original → docdb
GET https://ops.epo.org/rest-services/number-service/application/original/(PCT/GB02/04635).20021011/docdb
```

> **Tip:** Always include date, country code, and kind code separately when calling the number-service. Number formatting rules change over time.

---

## Number-service Status Codes

Returned in `ops:meta name="status"` element.

Key codes to know:

| Code | Severity | Meaning |
|------|----------|---------|
| pBRE001 | ERROR | Check digit wrong |
| pBRE002 | WARN | No country code |
| pBRE003 | WARN | Kind code not found |
| pBRE004 | WARN | Date out of range |
| pBRE012 | ERROR | Kind code mismatch |
| pBRE043 | ERROR | Not valid DOCDB format |
| pBRE999 | ERROR | Transformation failed, using original |
| BRE006 | ERROR | No matching pattern found |

See full table in [OPS Reference Guide](ops-reference-guide-v1.3.20.md) section 3.3.
