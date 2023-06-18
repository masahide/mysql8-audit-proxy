# mysql8-audit-proxy JSON ログ変換ツール

## 概要

mysql8-audit-proxy JSON ログ変換ツールは、mysql8-audit-proxy（MySQL バージョン8の監査ログを取得するプロキシ）が生成するバイナリログファイルを解析し、JSON形式に変換するユーティリティです。

## 特徴

このユーティリティは以下の機能を持っています：

- mysql8-audit-proxyが生成したgzipで圧縮されたバイナリログファイルの読み込みと解析
- タイムスタンプ、接続ID、ユーザー、データベース、アドレス、状態、エラー、コマンドなどのパケット情報をJSONに変換
- 生成されたJSONデータを標準出力に出力

## 使い方

### コマンドラインフラグ

- `-version`：ツールのバージョンを表示します。

### 引数

ユーティリティは引数として1つ以上のファイル名を受け取ります。これらは処理するgzipで圧縮されたバイナリログファイルです。

## インストール

ユーティリティをインストールするには、[リリースページ](https://github.com/masahide/mysql8-audit-proxy/releases)のAssetsからdebあるいはrpmファイルをダウンロードし、それを使用してインストールしてください。deb,rpmでインストールすると、ツールは`/usr/local/bin/mysql8-audit-log-decoder`にインストールされます。

## 使用例

ユーティリティを実行するコマンドの例は次のようになります：

```shell
$ /usr/local/bin/mysql8-audit-log-decoder <ファイル名>...
```
これにより、gzipで圧縮されたバイナリログファイルファイル名が処理され、結果のJSONが標準出力に出力されます。
