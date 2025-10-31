# LeakJS Pattern Files

This directory contains comprehensive regex pattern files for detecting various types of secrets and sensitive data in JavaScript files. These patterns are designed to minimize false positives while maximizing detection coverage.

## Available Pattern Files

### `payment-services.yaml`
Payment processor API keys and credentials:
- Stripe (live/test keys, webhooks, restricted keys)
- PayPal (client ID/secret)
- Braintree, Square, Adyen, 2Checkout, Checkout.com, Coinbase Commerce

### `cloud-providers.yaml`
Cloud platform credentials and tokens:
- AWS (access keys, session tokens, account IDs)
- Google Cloud (API keys, OAuth, service accounts)
- Azure (client secrets, storage keys, tenant IDs)
- DigitalOcean, Linode, Vultr, Heroku, Cloudflare, Firebase

### `communication-services.yaml`
Communication and collaboration platform tokens:
- Slack (tokens, webhooks)
- Discord (bot tokens, webhooks)
- Twilio (account SID, auth tokens, API keys)
- SendGrid, Mailgun, Postmark, SparkPost
- GitHub, GitLab, Bitbucket
- Twitter, Facebook, Instagram, LinkedIn, Telegram
- Microsoft Teams, Zoom

### `database-patterns.yaml`
Database connection strings and credentials:
- MongoDB, MySQL, PostgreSQL, Redis, SQLite
- Oracle, SQL Server, Elasticsearch, CouchDB
- InfluxDB, Cassandra, Neo4j, RabbitMQ
- Kafka, Memcached, DynamoDB, Firebase, Supabase
- PlanetScale, CockroachDB

### `crypto-patterns.yaml`
Cryptographic keys, tokens, and certificates:
- RSA/EC/DSA private keys, OpenSSH/PuTTY/PGP keys
- X.509 certificates, PKCS#12, JWT tokens
- OAuth/Bearer/API/Auth/Session tokens
- HMAC secrets, encryption/decryption keys
- AES/RSA/ECDSA/Ed25519 keys
- Salt/hash values, bcrypt/argon2/PBKDF2 hashes

### `personal-data.yaml`
Personally identifiable information (PII):
- Email addresses, phone numbers
- Credit card numbers (Visa, MasterCard, AmEx, Discover)
- Social Security Numbers, passport numbers, driver's licenses
- Bank account/routing numbers, IBAN, SWIFT codes
- IP addresses, MAC addresses, coordinates
- Postal codes, dates of birth, addresses
- Base64/hex encoded data, UUIDs

### `api-keys.yaml`
Various API service keys and tokens:
- OpenAI, Notion, Mapbox, Mailchimp
- Algolia, Airtable, Asana, Bitly
- Contentful, Datadog, Dropbox, Flickr
- HubSpot, Intercom, Jira, Last.fm
- New Relic, PagerDuty, Reddit, Rollbar
- Sentry, Shopify, SoundCloud, Spotify
- Trello, Travis CI, Twitch, Typeform
- Unsplash, Vimeo, Webflow, WordPress
- YouTube, Zendesk, Zoom

### `generic-patterns.yaml`
Generic patterns for common credential patterns:
- Generic API keys, secrets, tokens
- Environment variables, configuration values
- Database passwords, Redis/Memcached/RabbitMQ passwords
- SMTP/FTP/SSH passwords, admin/root passwords
- Service account keys, webhook secrets
- Signing secrets, verification tokens
- Client credentials, OAuth credentials
- Application secrets, private keys, certificates
- SSL/TLS certificates and keys

## Usage with LeakJS

Use these pattern files with LeakJS using the `-p` flag:

```bash
# Use payment service patterns
./leakjs -f myfile.js -p payment-services.yaml

# Use multiple pattern files
./leakjs -f myfile.js -p payment-services.yaml -p cloud-providers.yaml

# Use all patterns (built-in + custom)
./leakjs -f myfile.js -p payment-services.yaml -p cloud-providers.yaml -p communication-services.yaml
```

## Pattern Confidence Levels

- **High**: Very specific patterns with low false positive rate
- **Medium**: Moderately specific patterns, some potential false positives
- **Low**: Generic patterns that may match legitimate non-sensitive data

## Best Practices

1. **Start with specific patterns**: Use targeted pattern files based on the services you know are in use
2. **Combine patterns carefully**: Using too many generic patterns may increase false positives
3. **Review matches**: Always verify that detected patterns are actual secrets
4. **Use in CI/CD**: Integrate LeakJS with these patterns into your development pipeline
5. **Regular updates**: Keep pattern files updated as new secret formats emerge

## Contributing

When adding new patterns:
- Test extensively to ensure low false positive rates
- Use appropriate confidence levels
- Include comments explaining the pattern's purpose
- Follow the existing YAML structure
- Test with both positive and negative cases

## False Positive Mitigation

These patterns are designed to minimize false positives by:
- Using specific prefixes/suffixes where available
- Matching exact formats rather than generic strings
- Including length requirements where applicable
- Using word boundaries to avoid partial matches
- Focusing on known secret formats rather than guessing