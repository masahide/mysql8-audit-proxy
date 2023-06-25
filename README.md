# mysql8-audit-proxy

![semver](https://img.shields.io/github/v/tag/masahide/mysql8-audit-proxy)
![example workflow](https://github.com/masahide/mysql8-audit-proxy/actions/workflows/buildpkg.yml/badge.svg)
![gomod version](https://img.shields.io/github/go-mod/go-version/masahide/mysql8-audit-proxy/main)

`mysql8-audit-proxy` is a daemon that operates as a proxy between a MySQL version 8 server and a MySQL client. Its main function is to audit SQL operations coming from the client. It's licensed under the MIT License.

# Overview

The tool is written in Go and operates by listening for incoming MySQL client connections. When it receives SQL operation requests from the client, it generates an audit log of these operations. The log files are written in a unique binary format and are compressed using gzip. These files are automatically rotated based on a time interval specified by an environment variable.

To decode the compressed binary log files, a separate utility called [`mysql8-audit-log-decoder`](https://github.com/masahide/mysql8-audit-proxy/tree/main/cmd/mysql8-audit-log-decoder) is provided. This utility reads and parses the gzip-compressed binary log files generated by `mysql8-audit-proxy` and converts packet information such as timestamp, connection ID, user, database, address, state, error, and command into JSON format. The generated JSON data is then output to the standard output.

# Configuration Options
The tool can be configured using environment variables. Here are the default settings:

- `PROXY_LISTEN_ADDR`: The address that the proxy listens on. Default is `":3307"`.
- `PROXY_LISTEN_NET`: The network protocol used by the proxy. Default is `"tcp"`.
- `CON_TIMEOUT`: The connection timeout. Default is `"300s"`.
- `LOG_FILE_NAME`: The name format of the log file. Default is `"mysql-audit.%Y%m%d%H.log.gz"`.
- `ROTATE_TIME`: The time interval at which log files are rotated. Default is `"1h"`.
- `ADMIN_USER`: The admin user. Default is `"admin"`.
- `DEBUG`: Enable or disable debug mode. Default is `false`.


# Installation
Executables for `mysql8-audit-proxy` and `mysql8-audit-log-decoder` are created using Github Actions and are packaged into rpm and deb packages. These packages can be downloaded from the [Releases page](https://github.com/masahide/mysql8-audit-proxy/releases) and installed on your system.

# Usage
## mysql8-audit-proxy

To use mysql8-audit-proxy, you need to register the proxy connection information in advance. Specifically, you need to register connection usernames, connection passwords, etc., in the user table that mysql8-audit-proxy itself has. The settings are as follows:

- Start mysql8-audit-proxy
- Connect to mysql8-audit-proxy with the mysql command (using the admin user)
- Change the initial password of the admin user to any password
- Insert connection information into the user table

### Structure and Rules of the User Table
The user table of mysql8-audit-proxy consists of two columns: User and Password.

About the User column:
- The User column is in the form of `'<username>@<hostname[:port]>`
- `<username>` specifies the username to use on the MySQL server you are proxying to.
- `<hostname[:port]>` specifies the address (or hostname) and port number (port number is optional) of the MySQL server you are proxying to.
- Regular expressions can be used in the hostname. For example, if you set it to `user1@prd-.*`, it will apply to all servers with the `prd-` prefix.

About the Password column:
- The Password column specifies the password to use when connecting to the MySQL server you are proxying to.

Here's an example setup:

```bash
# Connect to mysql8-audit-proxy with the mysql command
MYSQL_PWD=pass  mysql -h 127.0.0.1 -P 3307 -uadmin user 
mysql> select * from user;
+-------+---------------------------------------------+
| User  | Password                                    |
+-------+---------------------------------------------+
| admin | y3aH0EuJdX2+R+0G5Wnmv4XBNBqQu8digBo1GypaA/Y |
+-------+---------------------------------------------+
1 rows in set (0.00 sec)

# Change the admin password
mysql> update user set Password='passxxx' where User='admin';

# Add server: `10.2.1.1`, user:`user1`, password:`passxxxxx`
mysql> insert user(User,Password) values('user1@10.2.1.1','passxxxxx');

# Add server: `prd-.*`, user:`root`, password:`Password00000`
mysql> insert user(User,Password) values('root@prd-.*','Password00000');
```

Example of Connection with MySQL Client
In the case of using the above User table setting example:
When using the MySQL client to connect via mysql8-audit-proxy running on localhost:3307, the command line will look like this:

```bash
MYSQL_PWD=passxxxxx mysql -h 127.0.0.1 -P 3307 -uuser1@10.2.1.1 db-name
```

## mysql8-audit-log-decoder
The [`mysql8-audit-log-decoder`](https://github.com/masahide/mysql8-audit-proxy/tree/main/cmd/mysql8-audit-log-decoder) utility can be used to convert the binary log files into JSON format. Pass the filename of the log file you want to process as an argument:
