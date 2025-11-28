# account-monitoring Specification

## Purpose
TBD - created by archiving change add-account-monitoring. Update Purpose after archive.
## Requirements
### Requirement: OKX API Authentication
The system SHALL authenticate with OKX API using API key, secret, and passphrase to access account data securely.

#### Scenario: Successful authentication
- **GIVEN** valid API credentials are configured
- **WHEN** the system makes an authenticated request to OKX API
- **THEN** the request includes proper signature headers (OK-ACCESS-KEY, OK-ACCESS-SIGN, OK-ACCESS-TIMESTAMP, OK-ACCESS-PASSPHRASE)
- **AND** the signature is computed as Base64(HMAC-SHA256(timestamp + method + path + body, secret))

#### Scenario: Invalid credentials
- **GIVEN** invalid API credentials are configured
- **WHEN** the system attempts to authenticate with OKX API
- **THEN** the system logs an authentication error
- **AND** the error is logged to the log file with ERROR level

#### Scenario: Missing credentials
- **GIVEN** API credentials are not configured
- **WHEN** the system starts
- **THEN** the system logs a configuration error
- **AND** the system exits gracefully with an error message

### Requirement: Account Balance Retrieval
The system SHALL fetch account balance data from OKX API endpoint GET /api/v5/account/balance at regular intervals.

#### Scenario: Successful balance retrieval
- **GIVEN** the system is running and authenticated
- **WHEN** the monitoring interval triggers (approximately every 60 seconds)
- **THEN** the system fetches balance data from OKX API
- **AND** parses the response to extract currency, balance, available, frozen, and equity fields
- **AND** the connection success is logged to the log file with INFO level

#### Scenario: API request failure
- **GIVEN** the system attempts to fetch balance data
- **WHEN** the OKX API request fails (network error, timeout, or HTTP error)
- **THEN** the system retries up to 3 times with exponential backoff (1s, 2s, 4s)
- **AND** logs each retry attempt with WARN level
- **AND** logs final failure with ERROR level if all retries fail

#### Scenario: Rate limit exceeded
- **GIVEN** the system makes requests to OKX API
- **WHEN** OKX API returns a rate limit error
- **THEN** the system backs off and waits for the next polling interval
- **AND** logs the rate limit error with WARN level

### Requirement: Position Data Retrieval
The system SHALL fetch open position data from OKX API endpoint GET /api/v5/account/positions at regular intervals.

#### Scenario: Successful position retrieval with open positions
- **GIVEN** the system is running and authenticated
- **WHEN** the monitoring interval triggers (approximately every 60 seconds)
- **THEN** the system fetches position data from OKX API
- **AND** parses the response to extract instrument, position_side, position_size, average_price, unrealized_pnl, margin, and leverage fields for each position
- **AND** the connection success is logged to the log file with INFO level

#### Scenario: No open positions
- **GIVEN** the system fetches position data
- **WHEN** the OKX API returns an empty positions list
- **THEN** the system logs "no open positions" with INFO level
- **AND** does not insert any position records to the database

#### Scenario: Position API request failure
- **GIVEN** the system attempts to fetch position data
- **WHEN** the OKX API request fails
- **THEN** the system retries up to 3 times with exponential backoff
- **AND** logs failures consistently with balance retrieval error handling

### Requirement: Data Persistence
The system SHALL persist fetched account balance and position data to a local SQLite database for historical analysis.

#### Scenario: Store balance snapshot
- **GIVEN** balance data is successfully retrieved from OKX API
- **WHEN** the data is ready to be stored
- **THEN** the system inserts a record into the account_balances table
- **AND** the record includes timestamp (UTC), currency, balance, available, frozen, and equity fields
- **AND** the timestamp is indexed for fast queries

#### Scenario: Store position snapshot
- **GIVEN** position data is successfully retrieved from OKX API
- **WHEN** the data contains one or more open positions
- **THEN** the system inserts records into the positions table for each position
- **AND** each record includes timestamp (UTC), instrument, position_side, position_size, average_price, unrealized_pnl, margin, and leverage fields
- **AND** the timestamp and instrument are indexed for fast queries

#### Scenario: Database write failure
- **GIVEN** the system attempts to write data to the database
- **WHEN** a database error occurs (disk full, corruption, connection lost)
- **THEN** the system logs the error with ERROR level
- **AND** the system exits gracefully to prevent data loss

#### Scenario: Database initialization
- **GIVEN** the system starts for the first time
- **WHEN** the database file does not exist
- **THEN** the system creates the database file at the configured path
- **AND** creates the account_balances and positions tables with proper schema
- **AND** logs successful initialization with INFO level

### Requirement: Monitoring Scheduler
The system SHALL run a continuous monitoring loop that fetches and stores account data at configurable intervals.

#### Scenario: Normal monitoring operation
- **GIVEN** the system is running
- **WHEN** the monitoring loop is active
- **THEN** the system fetches balance and position data approximately every 60 seconds (configurable)
- **AND** logs each monitoring cycle start with DEBUG level
- **AND** logs each monitoring cycle completion with INFO level

#### Scenario: Graceful shutdown
- **GIVEN** the system is running
- **WHEN** the system receives SIGINT or SIGTERM signal
- **THEN** the system completes the current monitoring cycle
- **AND** closes the database connection gracefully
- **AND** flushes log buffers
- **AND** logs shutdown event with INFO level
- **AND** exits with status code 0

#### Scenario: Startup health check
- **GIVEN** the system starts
- **WHEN** initialization is complete
- **THEN** the system performs a health check by fetching data from OKX API
- **AND** verifies database connectivity
- **AND** logs health check results with INFO level
- **AND** proceeds to monitoring loop only if health check passes

### Requirement: Configuration Management
The system SHALL load configuration from a YAML file and support template-based credential management to prevent accidental exposure.

#### Scenario: Load valid configuration
- **GIVEN** a valid config.yaml file exists with all required fields
- **WHEN** the system starts
- **THEN** the system loads API credentials (key, secret, passphrase)
- **AND** loads monitoring interval (default: 60 seconds)
- **AND** loads database path (default: ./data/tenyojubaku.db)
- **AND** loads log file path (default: ./logs/app.log)
- **AND** loads log level (default: INFO)
- **AND** logs configuration loaded successfully with INFO level (credentials masked)

#### Scenario: Missing configuration file
- **GIVEN** no config.yaml file exists
- **WHEN** the system starts
- **THEN** the system checks for config.template.yaml
- **AND** logs an error instructing the user to copy template to config.yaml
- **AND** exits with status code 1

#### Scenario: Invalid configuration format
- **GIVEN** config.yaml exists but has invalid YAML syntax
- **WHEN** the system attempts to load configuration
- **THEN** the system logs a parsing error with ERROR level
- **AND** exits with status code 1

### Requirement: Logging Infrastructure
The system SHALL log connection status, API interactions, and system events to rotating log files with appropriate log levels.

#### Scenario: Log connection success
- **GIVEN** the system successfully connects to OKX API
- **WHEN** data is retrieved
- **THEN** a log entry is written with INFO level
- **AND** the log entry includes timestamp, log level, and message "OKX API connection successful"

#### Scenario: Log connection failure
- **GIVEN** the system fails to connect to OKX API
- **WHEN** all retries are exhausted
- **THEN** a log entry is written with ERROR level
- **AND** the log entry includes timestamp, log level, error message, and error details

#### Scenario: Log API request/response
- **GIVEN** the system makes an API request
- **WHEN** the request completes
- **THEN** a log entry is written with DEBUG level
- **AND** the log entry includes timestamp, endpoint, HTTP status, and response time
- **AND** sensitive data (API keys, secrets) are masked in logs

#### Scenario: Log rotation
- **GIVEN** the log file grows beyond configured size (default: 100MB) or daily rotation is triggered
- **WHEN** rotation occurs
- **THEN** the current log file is renamed with timestamp suffix
- **AND** a new log file is created
- **AND** old log files are retained according to retention policy (default: 30 days)

#### Scenario: Log file creation
- **GIVEN** the system starts and log file does not exist
- **WHEN** the first log entry is written
- **THEN** the log directory is created if it doesn't exist
- **AND** the log file is created with appropriate permissions (read/write for owner only)

### Requirement: Error Recovery
The system SHALL implement retry logic and graceful degradation to handle transient failures without crashing.

#### Scenario: Transient network error recovery
- **GIVEN** the system encounters a network error during API request
- **WHEN** the error is transient (timeout, connection reset)
- **THEN** the system retries with exponential backoff (1s, 2s, 4s)
- **AND** logs each retry attempt
- **AND** continues normal operation if retry succeeds

#### Scenario: Persistent failure handling
- **GIVEN** the system encounters persistent failures (all retries exhausted)
- **WHEN** the failure continues across multiple monitoring cycles
- **THEN** the system continues monitoring (skips failed cycles)
- **AND** logs error details for debugging
- **AND** does not crash or exit

#### Scenario: Critical database error
- **GIVEN** the system encounters a critical database error (corruption, disk full)
- **WHEN** the error is detected
- **THEN** the system logs the critical error with ERROR level
- **AND** exits gracefully to prevent data corruption
- **AND** returns non-zero exit code

### Requirement: Security and Credential Protection
The system SHALL ensure that sensitive information (API credentials, logs) is never committed to the remote repository.

#### Scenario: Gitignore verification
- **GIVEN** the project repository exists
- **WHEN** the repository is initialized
- **THEN** .gitignore includes entries for config.yaml, logs/, and data/
- **AND** only config.template.yaml (with placeholder values) is committed
- **AND** README includes security warnings about credential management

#### Scenario: Credential validation
- **GIVEN** the system loads configuration
- **WHEN** API credentials are read
- **THEN** the system validates that credentials are not placeholder values
- **AND** warns if credentials appear to be defaults from template
- **AND** logs validation result

#### Scenario: Log data masking
- **GIVEN** the system logs API requests or configuration
- **WHEN** sensitive data is present (API keys, secrets, passphrases)
- **THEN** the sensitive data is masked with asterisks (e.g., "api_key: ab12****")
- **AND** only the first 4 characters are shown for debugging purposes

