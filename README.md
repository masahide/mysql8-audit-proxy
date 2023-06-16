# mysql8-audit-proxy
mysql8 audit proxy



# test

```bash
# start mysqld
docker run --rm --name testmysql -p3306:3306 -e MYSQL_ROOT_PASSWORD=PwTest01 -d mysql:8

# test connection
MYSQL_PWD=PwTest01 mysql -h 127.0.0.1 -P 3306 -uroot


# kill mysqld
docker kill testmysql

```

# dev test
```bash
# test
LISTEN_ADDRESS=:3307 go run main.go
MYSQL_PWD=pass  mysql -h 127.0.0.1 -P 3307 -uadmin  

```
