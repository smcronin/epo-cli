#!/usr/bin/env bash
# OPS API example queries — copy-paste ready (curl)
# Replace TOKEN with your Bearer token from https://developers.epo.org
# Register first to get Consumer Key + Secret, then:
#   TOKEN=$(curl -s -X POST https://ops.epo.org/3.2/auth/accesstoken \
#     -H "Authorization: Basic $(echo -n 'KEY:SECRET' | base64)" \
#     -H "Content-Type: application/x-www-form-urlencoded" \
#     -d "grant_type=client_credentials" | jq -r .access_token)

TOKEN="${EPO_TOKEN:-YOUR_BEARER_TOKEN}"
BASE="https://ops.epo.org/rest-services"
HDR=(-H "Authorization: Bearer $TOKEN" -H "Accept: application/json")

echo "=== BIBLIOGRAPHIC DATA ==="

# Single patent (JSON)
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000.A1/biblio" | jq .

# Without kind code (all publications for this application)
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000/biblio" | jq .

# With full lifecycle
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000.A1/biblio,full-cycle" | jq .

# Abstract only
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000/abstract" | jq .

# Bulk (GET, comma-separated)
curl -s "${HDR[@]}" \
  "$BASE/published-data/publication/epodoc/EP1676595,JP2006187606,US2006142694/biblio" | jq .

# Bulk (POST, up to 100)
curl -s "${HDR[@]}" -X POST \
  -H "Content-Type: text/plain" \
  "$BASE/published-data/publication/epodoc/biblio" \
  -d "EP1676595
JP2006187606
US2006142694" | jq .


echo "=== FULLTEXT ==="

# Check availability
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/published-data/publication/epodoc/EP1000000/fulltext" | jq .

# Claims
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/published-data/publication/epodoc/EP1000000/claims" | jq .

# Description
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/published-data/publication/epodoc/EP1000000/description" | jq .


echo "=== IMAGES ==="

# Image inquiry (get available links)
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000.A1/images" | jq .

# First page (PNG)
curl -s "${HDR[@]}" -H "Accept: image/png" \
  "$BASE/published-data/images/EP/1000000/PA/firstpage" -o /tmp/ep1000000_firstpage.png

# PDF page 1
curl -s "${HDR[@]}" -H "Accept: application/pdf" \
  -H "X-OPS-Range: 1" \
  "$BASE/published-data/images/EP/1000000/A1/fullimage" -o /tmp/ep1000000_p1.pdf


echo "=== FAMILY ==="

# INPADOC family (members only)
curl -s "${HDR[@]}" "$BASE/family/publication/docdb/EP.1000000.A1" | jq .

# Family with biblio
curl -s "${HDR[@]}" "$BASE/family/publication/docdb/EP.1000000.A1/biblio" | jq .

# Family with legal status
curl -s "${HDR[@]}" "$BASE/family/publication/docdb/EP.1000000.A1/legal" | jq .

# Family with both
curl -s "${HDR[@]}" "$BASE/family/publication/docdb/EP.1000000.A1/biblio,legal" | jq .

# By priority number
curl -s "${HDR[@]}" "$BASE/family/priority/docdb/US.18314305.A" | jq .

# Equivalents (simple family)
curl -s "${HDR[@]}" "$BASE/published-data/publication/epodoc/EP1000000/equivalents" | jq .


echo "=== NUMBER SERVICE ==="

# docdb → epodoc
curl -s "${HDR[@]}" \
  "$BASE/number-service/application/docdb/MD.20050130.A.20050130/epodoc" | jq .

# original → docdb (Japan)
curl -s "${HDR[@]}" \
  "$BASE/number-service/application/original/JP.%282006-147056%29.A.20060526/docdb" | jq .

# original → epodoc (USPTO)
curl -s "${HDR[@]}" \
  "$BASE/number-service/application/original/US.%2808%2F921%2C321%29.A.19970829/epodoc" | jq .

# PCT → docdb
curl -s "${HDR[@]}" \
  "$BASE/number-service/application/original/%28PCT%2FGB02%2F04635%29.20021011/docdb" | jq .


echo "=== REGISTER (EP only) ==="

# Bibliographic (default)
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/register/application/epodoc/EP99203729" | jq .

# Events
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/register/application/epodoc/EP99203729/events" | jq .

# Procedural steps
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/register/application/epodoc/EP99203729/procedural-steps" | jq .

# Combined
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/register/application/epodoc/EP99203729/biblio,events,procedural-steps" | jq .

# Search
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/register/search/?q=pa%3DIBM" | jq .


echo "=== LEGAL ==="

curl -s "${HDR[@]}" "$BASE/legal/publication/docdb/EP.1000000.A1" | jq .


echo "=== CLASSIFICATION (CPC) ==="

# Retrieve section A
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/classification/cpc/A" | jq .

# Specific class with depth
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/classification/cpc/H04W?depth=1" | jq .

# Search by keyword
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/classification/cpc/search/?q=machine+learning" | jq .

# ECLA → CPC mapping
curl -s "${HDR[@]}" -H "Accept: application/json" \
  "$BASE/classification/map/ecla/A61K9%2F00/cpc" | jq .


echo "=== SEARCH ==="

# Basic search
curl -s "${HDR[@]}" "$BASE/published-data/search?q=applicant%3DIBM" | jq .

# With biblio constituent
curl -s "${HDR[@]}" \
  "$BASE/published-data/search/biblio?q=applicant%3DIBM" | jq .

# With pagination
curl -s "${HDR[@]}" -H "X-OPS-Range: 26-50" \
  "$BASE/published-data/search?q=applicant%3DIBM" | jq .

# CPC + applicant + date range
curl -s "${HDR[@]}" \
  "$BASE/published-data/search?q=applicant%3DApple+and+cpc%3DH04W+and+pd+within+%2220230101+20231231%22" | jq .


echo "=== DATA USAGE ==="

curl -s -H "Authorization: Bearer $TOKEN" \
  "https://ops.epo.org/3.2/developers/me/stats/usage?timeRange=$(date +%d/%m/%Y)" | jq .
