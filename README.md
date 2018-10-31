# Antifreeze

[![Build Status](https://travis-ci.org/odlp/antifreeze.svg?branch=master)](https://travis-ci.org/odlp/antifreeze)

[Cloud Foundry](https://www.cloudfoundry.org/) CLI plugin to detect if an app has unexpected ENV vars or services bound which are missing from the manifest. Eliminate the snowflake!

Doubleplusgood with [Autopilot](https://github.com/concourse/autopilot), a CF CLI plugin for zero downtime application deploys, which demands an up-to-date manifest file.

## Installation

```
# For MacOS:
cf install-plugin https://github.com/odlp/antifreeze/releases/download/v0.4.0/antifreeze-darwin

# For Linux
cf install-plugin https://github.com/odlp/antifreeze/releases/download/v0.4.0/antifreeze-linux

# For Windows
cf install-plugin https://github.com/odlp/antifreeze/releases/download/v0.4.0/antifreeze.exe
```

Alternatively [download the latest release][latest-release] and place the binary on your path, e.g. `/usr/local/bin/antifreeze`.
Then run:

```sh
cf install-plugin /usr/local/bin/antifreeze
```

[latest-release]: https://github.com/odlp/antifreeze/releases/latest

Or if you have `go` installed:

```sh
go get -u github.com/odlp/antifreeze
cf install-plugin $GOPATH/bin/antifreeze
```

## Usage

```
cf check-manifest your-app-name -f manifest.yml
```

When your app has unexpected ENV vars or services you'll see output like this:

```
Running check-manifest...

App 'your-app-name' has unexpected ENV vars (missing from manifest ./manifest.yml):
- SNOW_FLAKE_VAR

App 'your-app-name' has unexpected services (missing from manifest ./manifest.yml):
- surprise-service
```

And the `check-manifest` command will exit with a non-zero status.

### Example with Autopilot

Your deployment script could include:

```sh
#!/bin/bash

set -e

cf check-manifest your-app-name -f manifest.yml
cf zero-downtime-push your-app-name -f manifest.yml

```


## Development

Clone the project & run the following:

```
./scripts/setup
```

Then you can run tests:

```
./scripts/test
```

And build & install locally to give any changes a spin:

```
./scripts/install-local
```
