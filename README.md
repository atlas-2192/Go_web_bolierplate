# Web Starter App

This is a starter web application. It is not a web framework. It is how I prefer to start web applications.

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
