# Project Context

## Purpose
[Describe your project's purpose and goals]

I need to design a complete trading system called TenyoJubaku (derived from the special ability "Ten-yū-jutsu-bō" in the anime *天與咒縛*). I want to add sufficient strict trading restrictions to my trading to minimize drawdowns and achieve stable compounding. The features I've currently designed include:

2025 Features:

1. Real-time monitoring and recording of trading account funds and position information (funds need to be stored in a database, as records are read approximately every minute, which is relatively frequent).

2. Automatically adding/completing stop-loss and take-profit orders if the position doesn't have set stop-loss and take-profit levels, or if the stop-loss/take-profit amount doesn't cover the entire position amount. The default stop-loss is set via a configuration file and is 1% of the volatility (not considering leverage; for 5x leverage, it's a 5% position loss stop-loss, and so on). The default take-profit is calculated based on the profit-loss ratio, which is also set through the configuration file. The default profit-loss ratio is 5:1 (without considering leverage; if the stop-loss is 1%, the profit-loss ratio is 5; with 5x leverage, it's 25% of the position for take-profit, and so on).

3. Order frequency limit, modified through the configuration file, defaults to a maximum of 5 orders per week. Market trading is prohibited; the trader is only allowed to act as a maker, not a taker (unless it's for take-profit, in which case partial position is allowed, with a default maximum of 50% for takers). When maker, the price difference must be at least 1% different from the market price (default configuration) to avoid FOMO (Fear of Missing Out). Multiple order confirmations are required; even if the order is successfully placed, a confirmation notification is sent every 12 hours, with a 4-hour waiting time. If the timeout occurs, the order amount is modified to 50% of the current amount (all configuration items).

Next Year Features:

4. Set up planned trading. This is to avoid missing some extreme market conditions. Since I mostly prefer left-side trading, I will list up to 3 price levels to capture possible spikes. The planned trades will be treated as special positions; this hasn't been designed yet, so I'll write about it later.

5. Order entry notes, including logic (text) and market summary and review (recorded voice, AI summarizes into text). This hasn't been designed yet, so I'll write about it later.

6. On-chain data acquisition and summary. This hasn't been designed yet, so I'll write about it later.

## Tech Stack
[List your primary technologies]

- Golang
- you can use any database, prefer lightweight ones like SQLite or similar
- you can use any script language if needed, but prefer python

## Project Conventions

### Code Style
[Describe your code style preferences, formatting rules, and naming conventions]

CamelCase naming conventions; variable names start with a lowercase letter, function names start with a capital letter.

You need to adhere to object-oriented design principles while maintaining loose coupling. 

The software needs to follow a layered design. 

Every function needs detailed comments (and the function brief should be in both Chinese and English, with only the brief in Chinese and the rest in English), and each part of the code within a function should also have a brief English comment. Commits for argument changes also need English comments. If there are any complex algorithms, detailed explanations in Chinese and English are required. Also, if function has return values, please explain each return value in the comments.

Maintain a consistent external interface, as front-end development may be needed in the future.

Configure files should be in YAML format.

### Architecture Patterns
[Document your architectural decisions and patterns]

Claude is currently running on macOS 14. It may be migrated to NAS later as the project matures.

### Testing Strategy
[Explain your testing approach and requirements]

Each layer requires sufficient unit tests to ensure security. 

Mid-layers can have fewer unit tests, but interface tests are essential. 

For top-level tests involving specific order placement or account transaction operations, a maximum of 5 USDT can be used as the test amount (as little as possible, just enough to meet the minimum order amount; if 5 USDT is insufficient, please contact me with the minimum order amount required for the transaction).

### Git Workflow
[Describe your branching strategy and commit conventions]

Each archive requires a commit to the remote repository. However, note that content containing user information must never be committed to the remote repository. If you are unsure whether a file contains user information, be sure to confirm with me.

create .gitignore and README.md file by yourself.

use git@github.com:wTHU1Ew/TenyoJubaku.git

## Domain Context
[Add domain-specific knowledge that AI assistants need to understand]

The API used in this project is provided by OKX Exchange(www.okx.com)

## Important Constraints
[List any technical, business, or regulatory constraints]

User information and specific configuration details must not be disclosed in the remote repository. The remote repository should only contain README.md and code information; configuration information should only be placed in .template files.

## External Dependencies
[Document key external services, APIs, or systems]

www.okx.com API has been download in ../document/markdown
