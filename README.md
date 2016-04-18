# Antifreeze

[Cloud Foundry](https://www.cloudfoundry.org/) plugin to detect if an app has unexpected ENV vars or services bound which are missing from the manifest. Eliminate the snowflake!

Doubleplusgood with [Autopilot](https://github.com/concourse/autopilot), a CF plugin for zero downtime application deploys, which demands an up-to-date manifest file.

## Installation

```
go get github.com/odlp/antifreeze
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

## Development

Clone the project & run the following:

```
./scripts/dev-setup.sh
```

Then you can run tests:

```
ginkgo
```

And build & install locally to give any changes a spin:

```
./scripts/install-local.sh
```
