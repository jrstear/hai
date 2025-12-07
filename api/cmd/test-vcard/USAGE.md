# Quick Test Guide

## Prerequisites

1. Elasticsearch must be running and accessible
2. Set `ELASTICSEARCH_URL` environment variable (or use `-es` flag)

## Quick Start

```bash
# From project root
cd api
go build -o ../bin/test-vcard ./cmd/test-vcard

# Test with sample vCard
export ELASTICSEARCH_URL="http://localhost:9200"
../bin/test-vcard -vcard ../test-data/sample-contacts.vcf
```

## Expected Behavior

The utility will:
1. Parse the vCard file
2. Store contacts in Elasticsearch `contacts` index
3. Read all contacts back from Elasticsearch
4. Compare imported vs stored data
5. Test individual contact retrieval

## Troubleshooting

**Error: "ELASTICSEARCH_URL environment variable or -es flag is required"**
- Set the environment variable: `export ELASTICSEARCH_URL="http://localhost:9200"`
- Or use the flag: `-es http://localhost:9200`

**Error: "Failed to create Elasticsearch client"**
- Check that Elasticsearch is running
- Verify the URL is correct
- Check network connectivity

**Error: "Failed to import vCard"**
- Verify the vCard file is valid
- Check file permissions
- Look for parsing errors in the output

## Sample vCard Format

The test utility expects standard vCard format (v3.0 or v4.0):

```
BEGIN:VCARD
VERSION:3.0
FN:John Doe
N:Doe;John;;;
EMAIL;TYPE=INTERNET:john@example.com
TEL;TYPE=CELL:+1-555-123-4567
END:VCARD
```









