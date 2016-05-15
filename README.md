# wijiwiki

A small wiki engine, written in Go.

- supports basic user authentication
- pages are written in markdown
- pages are cached in memory and the cache is refreshed on change
- can be run as standalone or as a FCGI process
- uses standard ``net/http``, no web frameworks

Missing features:

- Go dependency management (godep)
- templates (you have to create them by yourself)
