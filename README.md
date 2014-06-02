# Goship CLI interface

Command line interface for [Goship](https://github.com/gengo/goship/), deployment tool written in Go.

# Usage

Create `.goship.yml` config in your app dir:

```yaml
host: localhost:8000 # goship app host
project: navigator # project name
repo_owner: kirs # the same as in goship
repo_name: navigator_rails the same as in goship
user: kirs # your nickname
```

And then run:

```bash
goship-cli deploy production => deploy app to production
```
