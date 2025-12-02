# vCard Import Test Utility

A CLI utility to test vCard parsing and Elasticsearch storage.

## Usage

```bash
# Build the utility
cd api
go build -o ../bin/test-vcard ./cmd/test-vcard

# Run the test
export ELASTICSEARCH_URL="http://localhost:9200"
./bin/test-vcard -vcard path/to/contacts.vcf

# Or specify ES URL directly
./bin/test-vcard -vcard path/to/contacts.vcf -es http://localhost:9200
```

## What it does

1. **Imports vCard** - Parses the vCard file and stores contacts in Elasticsearch
2. **Lists contacts** - Reads all contacts back from Elasticsearch
3. **Compares results** - Verifies that imported contacts match stored contacts
4. **Tests retrieval** - Retrieves a single contact by ID to verify individual lookup

## Example Output

```
Testing vCard import:
  vCard file: test.vcf
  Elasticsearch: http://localhost:9200

Step 1: Importing vCard file...
  ✓ Imported 3 contact(s)

Imported contacts:
  [1] John Doe (ID: contact_abc123)
      Email: john@example.com
      Phone: +1-555-1234
      Name: John Doe
      External ID: vcf:abc123def456
      Source: vcf

Step 2: Reading contacts back from Elasticsearch...
  ✓ Found 3 total contact(s) in Elasticsearch

Step 3: Comparing imported vs stored contacts...
  ✓ Found imported contact: John Doe (ID: contact_abc123)
    ✓ All fields match

✓ SUCCESS: All 3 imported contact(s) were found in Elasticsearch

Step 4: Testing individual contact retrieval...
  ✓ Successfully retrieved contact: John Doe
  ✓ Contact data matches

Test complete!
```





