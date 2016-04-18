#!/bin/bash

cf uninstall-plugin AntifreezePlugin
go build
cf install-plugin ./antifreeze -f
