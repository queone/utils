# web
Based on [duckgo](https://github.com/sheepla/duckgo) and [ddgr](https://github.com/jarun/ddgr).
Just a CLI utility to query DuckDuckGo and open items from search result with Web browser quickly.

## Usage
```
Usage: duckgo [--json] [--timeout TIMEOUT] [--user-agent USER-AGENT] [--referrer REFERRER] [--browser BROWSER] [--version] [QUERY [QUERY ...]]

Positional arguments:
  QUERY                  keywords to search

Options:
  --json, -j             output results in JSON format
  --timeout TIMEOUT, -t TIMEOUT
                         timeout seconds
  --user-agent USER-AGENT, -u USER-AGENT
                         User-Agent value
  --referrer REFERRER, -r REFERRER
                         Referrer value
  --browser BROWSER, -b BROWSER
                         the command of Web browser to open URL
  --version              show version
  --help, -h             display this help and exit
```
