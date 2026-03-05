# European Publication Server REST Services Reference Guide

Source PDF: https://data.epo.org/publication-server/doc/EPS%20REST%20services.pdf

This file is a Markdown conversion of the EPO PDF document "European Publication Server REST services" (Revision 1.6, 2024/04).

## Revision Sheet

| Revision | Date | Description |
| --- | --- | --- |
| 1.6 | 2024/04 | REST service version 1.0 and 1.1 no longer supported; user-agent registration no longer necessary (all users have access to raw data in XML, PDF, ZIP, and HTML formats). |
| 1.5 | 2022/05 | Section 1.3: download limit increased to 10GB. |
| 1.4 | 2018/10 | Section 1.3: download limit increased to 5GB. |
| 1.3 | 2016/09 | Section 1.3: download limit increased to 4GB. |
| 1.2 | 2016/02 | Contact point changed; Section 2 improved URL responses; Section 4 clarified exchange of user-agent name with the EPO. |
| 1.1 | 2013/09 | New service in REST version 1.2 for image retrieval. |
| 1.0 | 2013/02 | Document creation. |

## Table of Contents

1. General information
2. Access to the REST service
3. List of EPS REST services
   - 3.1 Retrieval of publication dates
   - 3.2 Retrieval of weekly patent lists
   - 3.3 Retrieval of document formats
   - 3.4 Retrieval of raw data

## 1. General information

### 1.1 What is the European Publication Server

According to the European Patent Convention, the European Patent Office has the legal obligation to publish the patent applications it receives (A-documents) and the patents it grants (B-documents). The European Publication Server (EPS) (https://data.epo.org/publication-server/) has been the sole legally authoritative publication medium for European A and B documents since 1 April 2005.

### 1.2 What are REST services

REST defines a set of architectural principles by which you can design web services that focus on a system's resources, including how resource states are addressed and transferred over HTTP by a wide range of clients written in different languages.

### 1.3 REST services in the European Publication Server

The EPS REST API enables access to XML, HTML, TIFF images, and PDF/A of European A and B publications.

The fair-use policy limits a given IP address to 10GB of download in any sliding window of 7 days.

### 1.4 Contact point

For all matters relating to the European Publication Server and its REST services, contact `support@epo.org`.

## 2. Access to the REST service

The REST services are available at:

`https://data.epo.org/publication-server/rest/v1.2`

This URL shows the list of services (`publication-dates` and `patents`) supported by version 1.2.

Note: version 1.0 and 1.1 are no longer supported (the SOAP-based webservice is also no longer supported).

## 3. List of EPS REST services

### 3.1 Retrieval of publication dates

Description:

This service returns the list of all publication dates available in the EPS database.

URL template:

`https://data.epo.org/publication-server/rest/v1.2/publication-dates`

Request example:

```http
GET https://data.epo.org/publication-server/rest/v1.2/publication-dates
```

Response example:

```html
<html>
...
<body>
<a href="https://data.epo.org/.../publication-dates/20130904/patents">2013/09/04</a>
<a href="https://data.epo.org/.../publication-dates/20130911/patents">2013/09/11</a>
...
</body>
</html>
```

### 3.2 Retrieval of weekly patent lists

Description:

This service returns the list of patents being published at a given publication date.

URL template:

`https://data.epo.org/publication-server/rest/v1.2/publication-dates/{publicationDate}/patents`

Format of `{publicationDate}` is `YYYYMMDD`. Example: `20130904`.

Request example:

```http
GET https://data.epo.org/publication-server/rest/v1.2/publication-dates/20130904/patents
```

Response example:

```html
<html>
...
<body>
<a href="https://.../publication-dates/20130904/patents/EP1004359NWB1">EP1004359NWB1</a>
<a href="https://.../publication-dates/20130904/patents/EP1026129NWB1">EP1026129NWB1</a>
...
</body>
</html>
```

### 3.3 Retrieval of document formats

Description:

This service retrieves the list of available formats for a given document.

Formats available:

- XML
- HTML (transformation on-the-fly of the XML)
- PDF
- ZIP (including XML, PDF, and drawings in TIFF format)

Note that the availability of data in a given format may vary depending on the publication date.

URL template:

`https://data.epo.org/publication-server/rest/v1.2/patents/{patentNumber}`

Format of `{patentNumber}` is country-code + number + correction-code + kind-code. Example: `EP1004359NWB1`.

Request example:

```http
GET https://data.epo.org/publication-server/rest/v1.2/patents/EP1004359NWB1
```

Response example:

```html
<html>
<body>
<a href="https://data.epo.org/.../patents/EP1004359NWB1/document.xml">XML</a>
<a href="https://data.epo.org/.../patents/EP1004359NWB1/document.html">HTML</a>
<a href="https://data.epo.org/.../patents/EP1004359NWB1/document.pdf">PDF</a>
<a href="https://data.epo.org/.../patents/EP1004359NWB1/document.zip">ZIP</a>
</body>
</html>
```

### 3.4 Retrieval of raw data

Description:

This service provides access to raw data in XML, HTML, PDF, and ZIP formats.

URL template:

`https://data.epo.org/publication-server/rest/v1.2/patents/{patentNumber}/document.{format}`

Format of `{patentNumber}`: country-code + number + correction-code + kind-code. Example: `EP0729353NWB2`.

`{format}` is the preferred download format (`xml`, `html`, `pdf`, or `zip`) - see section 3.3.

Request example:

```http
GET https://data.epo.org/publication-server/rest/v1.2/patents/EP0729353NWB2/document.xml
```

Response example:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE ep-patent-document PUBLIC "-//EPO//EP PATENT DOCUMENT 1.4//EN" "ep-patent-document-v1-4.dtd">
<ep-patent-document id="EP95901961B2" file="EP95901961NWB2.xml" lang="en" country="EP" doc-number="0729353" kind="B2" date-publ="20120912" status="n" dtd-version="ep-patent-document-v1-4">
<SDOBI lang="en"><B000><eptags>
...
</ep-patent-document>
```
