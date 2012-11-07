## kensa-create-go

This repo is a Go template app for Heroku [kensa](http://github.com/heroku/kensa)
tool

```console
$ kensa create --template go
```

### Usage

```console
$ gem install kensa
$ gem install foreman

$ kensa create my-addon --template go
$ cd my-addon
$ go get
$ foreman start

$ cd my-addon
$ kensa test provision
$ tensa sso 1
```
