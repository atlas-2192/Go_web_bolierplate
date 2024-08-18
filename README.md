# Web Starter App

Warning: This is not ready for public use. It's not even close.

This is an experiment of how to build web applications and to make an example starter web application. It is not a web
framework. It is how I prefer to start web applications.

## Prerequisites

* A Unix-like environment - Linux, Mac, or WSL
* [Go](https://go.dev/) - Backend programming language
* [asdf](https://asdf-vm.com/) - Version manager for build tools like NodeJS and Ruby
* [direnv](https://direnv.net/) - Easily manage environment variables per directory
* [watchexec](https://watchexec.github.io/) - File system watcher used to trigger rebuilds
* [Templ](https://templ.guide/) - CLI for compiling HTML templates

## Stack

### Go

* [Cobra](https://github.com/spf13/cobra) - CLI parsing
* [Chi](https://github.com/go-chi/chi) - HTTP router
* [Templ](https://github.com/a-h/templ) - Compiled HTML templates
* [Zerolog](https://github.com/rs/zerolog) - Logger

## Setup

Initialize the config files:

```
rake setup:config
```

## Creating an new PostgreSQL Cluster

Ensure your `PATH` environment variable includes the PostgreSQL bin directory for the version of PostgreSQL you want to use. e.g. `/opt/homebrew/opt/postgresql@16/bin`.

Set the `PGPORT` environment variable if you want to run the server on a different port. Typically, this is done to allow multiple PostgreSQL clusters to run on the same host. This is best set in the `.envrc` file so subsequent development in this project will automatically connect to the proper cluster.

Create a new cluster:

```
rake setup:create_postgresql_cluster
```

The cluster will be created in the `.postgresql` directory and is setup to log all queries to `stderr`.

You can run the database with:

```
rake db:cluster:run
```

## Setup PostgreSQL Database and User

Once the new cluster is running (or you skipped that step and are using an existing, running cluster) setup the database
and user:

```
rake setup:postgresql
```
