## kensa-create-go

A add-on template app in Go, for use with [kensa](http://github.com/heroku/kensa).

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
