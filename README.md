# Plex

[![GoTemplate](https://img.shields.io/badge/go/template-black?logo=go)](https://github.com/SchwarzIT/go-template)

toy web browser

The project uses `make` to make your life easier. If you're not familiar with Makefiles you can take a look at [this quickstart guide](https://makefiletutorial.com).

Whenever you need help regarding the available actions, just use the following command.

```bash
make help
```

## Setup

To get your setup up and running the only thing you have to do is

```bash
make all
```

This will initialize a git repo, download the dependencies in the latest versions and install all needed tools.
If needed code generation will be triggered in this target as well.

## Test & lint

Run linting

```bash
make lint
```

Run tests

HTML tokenizer TEST files

`TEST_HTML_TOKENIZER_SUBSET` a comma sperated list of file names, see testdata folder in html_tokenizer 


```bash
make test
```
