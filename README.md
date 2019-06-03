# Pacm

Simple package manager for managing binary applications. A simple ini config is 
used to declare what packages should be installed and managed. `pacm` won't 
attempt to add itself to your PATH, this will need to be done manually.

`pacm` works with a single config file. This config file can declare packages and 
recipes. A `recipe` is a definition for where to download the `recipe` from and
how to extract the `recipe`. A `package` is a combination of `recipe` and a version
of that recipe: `<recipe>@<version`. These can be defined in your config file.

## Recipes

Some default ones can be found at https://github.com/vishen/pacm-recipes.

New recipes can be added to your config file and will take precendence
over recipes from any remote recipes.

## Config

Will by default look for a config path at `~/.config/pacm/config`.

```ini
dir=/some/path/on/disk

[terraform@0.12.0-rc1]
	active=true
[terraform@0.11.13]

[protoc@3.8.0-rc1]
	active=true
[protoc@3.7.1]

[kubectl@1.15.0-alpha.1]
[kubectl@1.14.2]
	active=true
```

`dir` indicates which directory to install binaries to. This will
need to be added to your path.

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
  clean        Clean up cached archives
  ensure       Ensure that your binaries are up-to-date
  help         Help about any command
  list-updates Available updates for installed package
  status       Status of installed packages
  update       Update packages

Flags:
  -f, --config string   pacm config file to load (defaults to ~/.config/pacm/config)
  -h, --help            help for pacm
  -x, --log-commands    log commands being run

Use "pacm [command] --help" for more information about a command.
```

## Running

Make sure that you have added the `dir` path to you PATH, otherwise
you won't have the installed binaries available to you.

When running `pacm activate` and `pacm update`, your ini config
will be overwritten to reflect the changes you have made.

```
# Ensure that your config file and what is installed on disk is correct.
$ pacm ensure

$ terraform version
Terraform v0.12.0-rc1

Your version of Terraform is out of date! The latest version
is 0.12.0. You can update by downloading from www.terraform.io/downloads.html

# Active a package <recipe>@<version>. 
# NOTE: The package needs to be in your config file (otherwise use pacm update).
$ pacm activate terraform@0.11.13

$ terraform version
Terraform v0.11.13

Your version of Terraform is out of date! The latest version
is 0.12.0. You can update by downloading from www.terraform.io/downloads.html

# List updates from github about a recipe.
$ pacm list-updates terraform
+--------------------------+------------------+--------------+-------------------------------+
|           TAG            |      STATUS      | GITHUBSTATUS |           PUBLISHED           |
+--------------------------+------------------+--------------+-------------------------------+
| terraform@v0.12.0        |                  |              | 2019-05-22 20:24:00 +0000 UTC |
| terraform@v0.12.0-rc1    | installed        | pre-release  | 2019-02-28 22:59:41 +0000 UTC |
| terraform@v0.12.0-alpha1 |                  | pre-release  | 2018-10-20 00:49:09 +0000 UTC |
| terraform@v0.11.14       |                  |              | 2019-05-16 20:49:01 +0000 UTC |
| terraform@v0.11.13       | active,installed |              | 2019-03-11 18:51:43 +0000 UTC |
| terraform@v0.11.12-beta1 |                  | pre-release  | 2019-01-28 12:39:21 +0000 UTC |
| terraform@v0.11.12       |                  |              | 2019-03-08 20:09:11 +0000 UTC |
| terraform@v0.11.10       |                  |              | 2018-10-23 15:06:04 +0000 UTC |
| terraform@v0.11.9-beta1  |                  | pre-release  | 2018-10-15 20:24:38 +0000 UTC |
| terraform@v0.11.0-rc1    |                  | pre-release  | 2017-11-09 19:59:50 +0000 UTC |
| terraform@v0.11.0-beta1  |                  | pre-release  | 2017-11-03 23:48:26 +0000 UTC |
| terraform@v0.11.0        |                  |              | 2017-11-16 19:34:52 +0000 UTC |
+--------------------------+------------------+--------------+-------------------------------+

# Update a recipe.
$ pacm update terraform@0.12.0

$ terraform version
Terraform v0.12.0

$ pacm list-updates terraform
+--------------------------+------------------+--------------+-------------------------------+
|           TAG            |      STATUS      | GITHUBSTATUS |           PUBLISHED           |
+--------------------------+------------------+--------------+-------------------------------+
| terraform@v0.12.0        | active,installed |              | 2019-05-22 20:24:00 +0000 UTC |
| terraform@v0.12.0-rc1    | installed        | pre-release  | 2019-02-28 22:59:41 +0000 UTC |
| terraform@v0.12.0-alpha1 |                  | pre-release  | 2018-10-20 00:49:09 +0000 UTC |
| terraform@v0.11.14       |                  |              | 2019-05-16 20:49:01 +0000 UTC |
| terraform@v0.11.13       | installed        |              | 2019-03-11 18:51:43 +0000 UTC |
| terraform@v0.11.12-beta1 |                  | pre-release  | 2019-01-28 12:39:21 +0000 UTC |
| terraform@v0.11.12       |                  |              | 2019-03-08 20:09:11 +0000 UTC |
| terraform@v0.11.10       |                  |              | 2018-10-23 15:06:04 +0000 UTC |
| terraform@v0.11.9-beta1  |                  | pre-release  | 2018-10-15 20:24:38 +0000 UTC |
| terraform@v0.11.0-rc1    |                  | pre-release  | 2017-11-09 19:59:50 +0000 UTC |
| terraform@v0.11.0-beta1  |                  | pre-release  | 2017-11-03 23:48:26 +0000 UTC |
| terraform@v0.11.0        |                  |              | 2017-11-16 19:34:52 +0000 UTC |
+--------------------------+------------------+--------------+-------------------------------+
```
