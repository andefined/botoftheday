[![Build Status](https://travis-ci.org/andefined/botoftheday.svg?branch=master)](https://travis-ci.org/andefined/botoftheday)
[![Go Report Card](https://goreportcard.com/badge/github.com/andefined/botoftheday)](https://goreportcard.com/report/github.com/andefined/botoftheday)
[![Go Releaser](https://img.shields.io/github/release/andefined/botoftheday.svg)](https://goreportcard.com/report/github.com/plagiari-sm/andefined/botoftheday/latest)
[![Code Coverage](https://img.shields.io/codecov/c/github/andefined/botoftheday/master.svg)](https://codecov.io/gh/andefined/botoftheday/releases/latest)

##### botoftheday
> Just a Bot that monitors other Bots ++

## Bot of the Day
The source code of [@BotOfTheDay_](https://twitter.com/BotOfTheDay_).

Twitter **Bots** are a major issue nowadays. **Trump** won the elections because of Bots. This CLI simple streams data for a given **User** and generates a couple of basic charts (Activity, Mentions) regarding the User and his/her Bot.

#### Config Template

I am using 2 Twitter application because the one that **posts** the tweets, might get blocked.
```yaml
stream:
    consumer-key: xxx
    consumer-secret: xxx
    access-token: xxx-xxx
    access-token-secret: xxx
list:
    consumer-key: xxx
    consumer-secret: xxx
    access-token: xxx-xxx
    access-token-secret: xxx
```

#### Pipeline
```bash
# Stream Data
botoftheday -conf conf/bot.yaml -path path/to/folder/ -user username stream
# Create the daily output
botoftheday -conf conf/bot.yaml -path path/to/folder/ -user username list
# Generate the charts (.png)
botoftheday -conf conf/bot.yaml -path path/to/folder/ -user username generate
# Post on twitter
botoftheday -conf conf/bot.yaml -path path/to/folder/ -user username post
```
