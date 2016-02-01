# [DEPRECATED]

This tool is deprecated. Please follow [this guide](https://devcenter.heroku.com/articles/building-a-heroku-add-on) when building a Heroku add-on.

## kensa-create-go

An add-on template app in Go, for use with [kensa](http://github.com/heroku/kensa).

### Usage

```console
$ gem install kensa
$ gem install foreman

$ kensa create my-addon --template go
$ cd my-addon
$ go get
$ echo "web: my-addon" > Procfile
$ foreman start

$ cd my-addon
$ kensa test all
```
