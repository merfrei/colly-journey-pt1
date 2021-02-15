Colly Journey PT 1
==================

For learning purposes.

A Colly journey building some web crawlers and extracting data from the web using the [Colly Scraping Framework](http://go-colly.org/).


## Trying it

**Checking my config**

```shell
~$ cat mongo.conf | grep port
  port: 22692
```

Create and edit `config.toml`

```shell
~$ cp config.toml.example config.toml
```

```toml
[mongo]
uri = "mongodb://localhost:22692"
database = "factchecks"
```

**Starting MongoDB** (you should install it before)

```shell
~$ mkdir -p data/mongodb

~$ mongod -f mongo.conf
```

**Running spiders example**

```shell
~$ go run cmd/politifact/politifact.go
```
