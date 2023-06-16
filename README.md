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
MYSQL_PWD=pass  mysql -h 127.0.0.1 -P 3307 -uadmin user -e "select * from user"


```bash
MYSQL_PWD=pass  mysql -h 127.0.0.1 -P 3307 -uadmin user 
```


# Usage

Check and change the password information of the administrative user of mysql8-audit-proxy
```sql
mysql> select * from user;
+---------+----------+
| User    | Password |
+---------+----------+
| admin   | pass     |
+---------+----------+
1 rows in set (0.00 sec)

mysql> update user set Password='passxxx' where User='admin';
```

## Register the target mysql server and user/password. 
For the server name and user/password information of the proxy destination mysql server, connect to mysql8-audit-proxy as an admin user and insert it into the user table.

```sql
insert user(User,Password) values('<username>@<hostname[:port]>', '<password>');
insert user(User,Password) values('<username>@<Regular expression>', '<password>');
```
### example
```sql
mysql> insert user(User,Password) values('user1@10.2.1.1','passxxxxx');
mysql> insert user(User,Password) values('root@.*','Password00000');
```

## Connect to mysql via mysql8-audit-proxy

```bash
MYSQL_PWD=<password> mysql -h 127.0.0.1 -P 3307 -u<username>@<hostname[:port]> <dbname>
# example
MYSQL_PWD=passxxxxx> mysql -h 127.0.0.1 -P 3307 -uuser1@10.2.1.1 dbname

```

