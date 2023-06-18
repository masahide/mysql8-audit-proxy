# mysql8-audit-proxy JSON Log Converter Tool

## Overview

The mysql8-audit-proxy JSON Log Converter Tool is a utility that parses binary log files generated by mysql8-audit-proxy (a proxy for auditing logs in MySQL version 8) and converts them into JSON format.

## Features

This utility provides the following features:

- Reads and parses gzip-compressed binary log files generated by mysql8-audit-proxy
- Converts packet information such as timestamp, connection ID, user, database, address, state, error, and command into JSON format
- Outputs the generated JSON data to the standard output

## Usage

### Command Line Flag

- `-version`: Displays the tool's version.

### Arguments

The utility accepts one or more filenames as arguments. These are the gzip-compressed binary log files to be processed.

## Installation

To install the utility, download the deb or rpm file from the Assets on the [Release Page](https://github.com/masahide/mysql8-audit-proxy/releases) and install it using that file. When installed via deb or rpm, the tool is installed at `/usr/local/bin/mysql8-audit-log-decoder`.

## Example of Use

An example command to run the utility is as follows:

```shell
$ /usr/local/bin/mysql8-audit-log-decoder filename
```

This processes the gzip-compressed binary log file filename and outputs the resulting JSON to the standard output.