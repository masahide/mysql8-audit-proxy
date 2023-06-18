# Contributing to mysql8-audit-proxy

## Introduction
Thank you for your interest in contributing to mysql8-audit-proxy! This project is a utility that helps with auditing logs in MySQL version 8. We appreciate all contributions and are always looking for help to improve the project.

## How to Contribute
We welcome contributions in all forms. Here's how to get started:

1. Fork the repository on GitHub.
2. Clone your fork and create a new branch for your feature or fix.
3. Write the code for your feature or fix.
4. Write tests to accompany your code.
5. Push your branch to your fork.
6. Open a pull request in the main repository.

Please adhere to our coding style guidelines. 

## Reporting Issues
If you find a bug or issue with mysql8-audit-proxy, please open an issue on the GitHub repository. Please include as much detail as possible, including steps to reproduce the issue, any error messages, and information about your operating system and version of MySQL.

## Feature Requests
We're always open to new ideas! If you have a suggestion for a new feature, please open an issue on the GitHub repository and label it as a feature request.

## Getting Help
If you need help with using mysql8-audit-proxy, please open an issue on the GitHub repository. We'll do our best to assist you!


### Start and Stop MySQL Daemon
```bash
# Start mysqld
docker run --rm --name testmysql -p3306:3306 -e MYSQL_ROOT_PASSWORD=PwTest01 -d mysql:8

# Test connection
MYSQL_PWD=PwTest01 mysql -h 127.0.0.1 -P 3306 -uroot

# Kill mysqld
docker kill testmysql
```

## Test the Program

```bash
# Proxy admin access 
LISTEN_ADDRESS=:3307 go run main.go
MYSQL_PWD=pass  mysql -h 127.0.0.1 -P 3307 -uadmin user -e "select * from user"

# MySQL proxy test
MYSQL_PWD=PwTest01 mysql -h 127.0.0.1 -P 3307 -uroot@127.0.0.1 mysql
```

We're excited to see your contributions!
