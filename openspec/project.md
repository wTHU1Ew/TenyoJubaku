# Project Context

## Purpose
[Describe your project's purpose and goals]

I need to design a complete trading system called TenyoJubaku (derived from the special ability "Ten-yū-jutsu-bō" in the anime *天與咒縛*). I want to add sufficient strict trading restrictions to my trading to minimize drawdowns and achieve stable compounding. The features I've currently designed include:

2025 Features:

1. Real-time monitoring and recording of trading account funds and position information (funds need to be stored in a database, as records are read approximately every minute, which is relatively frequent).

2. If no stop-loss or take-profit level is set for the position, or if the stop-loss/take-profit amount is insufficient to cover the entire position, stop-loss and take-profit orders will be automatically added/completed. The default stop-loss level is set via configuration file and is 1% of volatility (for a long position, this means a 1% drop in the opening price is set as the stop-loss). The default take-profit level is calculated based on the risk-reward ratio, which is also set via configuration file. The default risk-reward ratio is 5:1 (for a long position, this means a 5% increase in the opening price is set as the take-profit).

3. Order frequency limit, modified through the configuration file, defaults to a maximum of 5 orders per week. Market trading is prohibited; the trader is only allowed to act as a maker, not a taker (unless it's for take-profit, in which case partial position is allowed, with a default maximum of 50% for takers). When maker, the price difference must be at least 1% different from the market price (default configuration) to avoid FOMO (Fear of Missing Out). Multiple order confirmations are required; even if the order is successfully placed, a confirmation notification is sent every 12 hours, with a 4-hour waiting time. If the timeout occurs, the order amount is modified to 50% of the current amount (all configuration items).

4. some cli feature

Next Year Features:

5. Set up planned trading. This is to avoid missing some extreme market conditions. Since I mostly prefer left-side trading, I will list up to 3 price levels to capture possible spikes. 通常我会设置3个买入点位，分别是极端左侧、中间左侧和轻微左侧。每个点位的买入量可以通过配置文件设置，默认情况下，极端左侧买入80%，中间左侧买入50%，轻微左侧买入30%。极端左侧，中间左侧，轻微左侧理论上不会相互覆盖，以多单左侧交易举例，只有轻微左侧的止损被触发之后，之后价格再继续往下降才会到中间左侧的开单位置，只有中间左侧的止损被触发之后，之后价格再继续往下降才会到极端左侧的开单位置。止损逻辑默认为feature1的tpsl，除非手动设置了止损且覆盖全仓位，否则剩余仓位都会由tpsl按照配置文件设置默认止损。当价格达到某个点位时，系统会自动下单买入相应的数量。如果价格没有达到任何一个点位，则不会进行任何操作。如果只设置了一个order点位，则默认为轻微左侧，买入30%。如果设置了两个order点位，则默认为中间左侧买入50%和轻微左侧买入30%，如果设置了三个点位则为极端左侧，中间左侧，轻微左侧。当然也可以在cli下单的时候，通过启动参数强制要求当前下单的为极端左侧，中间左侧，轻微左其中一种。除此之外，还有一个很重要的交易策略为动态tpsl，以多头左侧为例，当当前价格相较于开单价格上涨超过1%（通过配置文件里的firstMove配置），则自动将止损设置为开单价格 *（1+0.001），这里的0.001是为了cover住交易手续费。如果价格继续上涨，则止损位置也会继续上移，出发firstMove之后，价格每上涨0.5%，则止损价格上移0.1%。当价格回落到止损位置时，全部仓位止损离场。当价格到达止盈位置时，全部仓位止盈离场。

6. Order entry notes, including logic (text) and market summary and review (recorded voice, AI summarizes into text). This hasn't been designed yet, so I'll write about it later.

7. On-chain data acquisition and summary. This hasn't been designed yet, so I'll write about it later.

## Tech Stack
[List your primary technologies]

- Golang
- you can use any database, prefer lightweight ones like SQLite or similar
- you can use any script language if needed, but prefer python
- don't use any third-party paid services
- avoid direct database operations in Go; use ORM libraries instead

## Project Conventions

### Code Style
[Describe your code style preferences, formatting rules, and naming conventions]

CamelCase naming conventions; variable names start with a lowercase letter, function names start with a capital letter.

You need to adhere to object-oriented design principles while maintaining loose coupling. 

The software needs to follow a layered design. 

Every function needs detailed comments (and the function brief should be in both Chinese and English, with only the brief in Chinese and the rest in English), and each part of the code within a function should also have a brief English comment. Commits for argument changes also need English comments. If there are any complex algorithms, detailed explanations in Chinese and English are required. Also, if function has return values, please explain each return value in the comments.

Maintain a consistent external interface, as front-end development may be needed in the future.

Configure files should be in YAML format.

Project progress will be recorded in the `docs` directory, allowing AI to read the `.md` files within `docs` to obtain project progress. AI will also record its work in the `docs` directory.

- Organized all documentation into `docs/features/` by feature category

- Created feature-specific folders: feature1-tpsl, feature2-position-management, feature3-order-control, infrastructure, architecture, archived

- Established naming convention: `<NAME>_<TYPE>_V<VERSION>_<DATE>.md`

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

you should do "git" to push arhcieved changes to remote resposity.

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
