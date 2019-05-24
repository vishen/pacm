# Pacm

Simple package manager for managing binary applications. A
simple ini config is used to declare what packages should be
installed and managed. `pacm` won't attempt to add itself to
your PATH, this will need to be done manually.


## Recipes

Can be found at https://github.com/vishen/pacm-recipes.

## Config

```ini
dir=/some/path/on/disk

[terraform@0.12.0-rc1]
	active=true
[terraform@0.11.11]

[protoc@3.8.0-rc1]
	active=true
[protoc@3.7.1]

[kubectl@1.15.0-alpha.1]
[kubectl@1.14.2]
	active=true
```

`dir` indicates which directory to install binaries to.

`[<recipe>@<version>]` where `recipe` is a `[recipe <name>]` declared
somewhere in a config file (ie: https://github.com/vishen/pacm-recipes/blob/e22e9659bfdaee20ade7a1654753c05a41597426/kubectl/recipe.ini).

## Installing

	go get -u github.com/vishen/pacm

## Commands

```
Simple package manager for binaries

Usage:
  pacm [command]

Available Commands:
  activate     Activate packages
  ensure       Ensure that your binaries are up-to-date
  help         Help about any command
  list-updates Available updates for installed package
  status       Status of installed packages
  update       Update packages

Flags:
  -f, --config string   pacm config file to load (defaults to ~/.config/pacm/config)
  -h, --help            help for pacm

Use "pacm [command] --help" for more information about a command.
```
