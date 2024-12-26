package capsule

// データベースに小規模 〜 中規模の JSON データを保存するための関数
// MySQL で扱うデータの容量制限は一般的には以下の通り
// データベース: なし > テーブル: 64TB > レコード（行）: 64KB > フィールド: 合計でレコードのサイズ以内
// MySQL の行サイズは 64KB なのでたとえば 50,000 字の JSON データはそのままレコードに収まらない

// しかし、LONGTEXT や BLOB など巨大なデータ型を扱う場合、フィールドにはそのデータのポインタのみ置かれる
// JSON 型の扱いは LONGTEXT 型に準ずるのでこのままのやり方で 4GB くらいまでは保存できるらしい
// MariaDB は MySQL の派生なのでおそらく大体おなじ

import (
	"fmt"
	"log"
	"os"
	"time"

	json "encoding/json"

	"github.com/fatih/color"
	mysql "github.com/go-sql-driver/mysql" // MariaDB を使う場合もこの MySQL ドライバが使用可能
	sqlx "github.com/jmoiron/sqlx"
	godotenv "github.com/joho/godotenv"
)

var Db *sqlx.DB

func init() {
	godotenv.Load(".env")
	// .env から環境変数の設定を読み込む
	// .env は Git 管理外なので NeoShowcase の環境には存在せずこのコードはエラーになるが、
	// NeoShowcase ではもとから環境変数が設定されているのでエラーをスルーして処理を続行
}

func Connect() {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Println(color.HiYellowString("[failed to load location] %s", err))
		panic(color.HiRedString("[failed to initialize database]"))
	}

	conf := mysql.Config{ // .env から読み込んだ環境変数をもとにデータベースを定義
		User:                 os.Getenv("NS_MARIADB_USER"),
		Passwd:               os.Getenv("NS_MARIADB_PASSWORD"),
		Net:                  "tcp",
		Addr:                 os.Getenv("NS_MARIADB_HOSTNAME") + ":" + os.Getenv("NS_MARIADB_PORT"),
		DBName:               os.Getenv("NS_MARIADB_DATABASE"),
		ParseTime:            true,
		Collation:            "utf8mb4_unicode_ci",
		AllowNativePasswords: true, // これがないとパスワード認証で突っぱねられる
		Loc:                  jst,
	}
	// 本来ならこの storage を環境独立にするために環境変数も引数として受け取った方が良いだろうけど、
	// データベースを使うなら大概 NeoShowcase だろうという甘い読みで引数にはしていない

	if Db, err = sqlx.Open("mysql", conf.FormatDSN()); err != nil { // データベースに接続
		panic(color.HiRedString("[failed to open database] %s", err))
	}
	log.Println(color.GreenString("[connected to database]"))
}

func SetUp[T any](origin T, reset bool) {
	// 引数はデータベースに保存するデータの初期値のポインタ
	// データベースに何も保存されていない最初の状態や異常時にのみこの値を用いる

	Connect() // データベースとの接続だけを切り出した関数

	if _, err := Db.Exec(`CREATE TABLE IF NOT EXISTS config (json JSON);`); err != nil {
		log.Println(color.HiYellowString("[failed to create table] %s", err))
		panic(color.HiRedString("[failed to initialize table] make sure your container is ready!"))
	}

	count := 0 // すでに存在するレコードの数
	if err := Db.Get(&count, `SELECT COUNT(*) FROM config`); err != nil {
		log.Println(color.HiYellowString("[failed to get count of table] %s", err))
		panic(color.HiRedString("[failed to initialize table]"))
	}

	// reset = true または count = 0 などのときにはテーブル config を初期化する
	if (count != 1) || reset {
		if _, err := Db.Exec(`TRUNCATE TABLE config`); err != nil { // テーブルを空にする
			panic(color.HiRedString("[failed to empty table] %s", err))
		}

		if _, err := Db.Exec(`INSERT INTO config (json) VALUES ('{}')`); err != nil {
			panic(color.HiRedString("[failed to insert record into table] %s", err))
		}

		if err := Save(origin); err != nil {
			panic(color.HiRedString("[failed to save original data] %s", err))
		}
	}

	log.Println(color.GreenString("[initialized table]"))
}

func Save[T any](config T) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal %v: %w", config, err)
	}
	// json.Marshal(config) は構造体 config を JSON 形式のテキストにする

	if _, err = Db.Exec(`UPDATE config SET json = ?`, string(configJson)); err != nil {
		return fmt.Errorf("failed to update the database: %w", err)
	}
	// UPDATE文は（WHERE 以下の条件にあてはまる）全てのレコードを書き換える。レコードは一つしかないので WHERE 文は不要
	// データベース bot_db > テーブル config > レコード x 1 > フィールド json

	return nil // 必ず error 型の返り値を返す必要があるので nil を返す（nil は error 型か…？）
}

func Load[T any]() (T, error) {
	record := struct { // データベースに保存されているレコードを受け取るための型
		Json string `json:"json"`
	}{}

	var config T // エラーの場合の返り値

	if err := Db.Get(&record, "SELECT * FROM config"); err != nil {
		return config, fmt.Errorf("failed to get data from database: %w", err)
	}
	// record にデータベースのレコードの値を写し取って、

	if err := json.Unmarshal([]byte(record.Json), &config); err != nil {
		return config, fmt.Errorf("failed to unmarshal %s: %w", record.Json, err)
	}
	// JSON をデコードして config に代入

	return config, nil
}

func With[T any](action func(config *T) error) error {
	conf, err := Load[T]()
	if err != nil {
		return fmt.Errorf("in with: %w", err)
	}

	if err := action(&conf); err != nil {
		return err
	} // 実行する関数そのものを引数に渡してソースコードをシンプルにする

	if err := Save(conf); err != nil {
		return fmt.Errorf("in with: %w", err)
	}
	return nil
}
