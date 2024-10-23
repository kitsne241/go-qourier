package storage

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

	mysql "github.com/go-sql-driver/mysql" // MariaDB を使う場合もこの MySQL ドライバが使用可能
	sqlx "github.com/jmoiron/sqlx"
	godotenv "github.com/joho/godotenv"
)

type Capsule struct {
	Db *sqlx.DB
}

var cps Capsule

func init() {
	godotenv.Load(".env")
	// .env から環境変数の設定を読み込む
	// .env は Git 管理外なので NeoShowcase の環境には存在せずこのコードはエラーになるが、
	// NeoShowcase ではもとから環境変数が設定されているのでエラーをスルーして処理を続行
}

func SetUp() error {
	jst, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return fmt.Errorf("failed to load location: %w", err)
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
	// 本来ならこの courier を環境独立にするために環境変数も引数として受け取った方が良いだろうけど、
	// データベースを使うなら大概 NeoShowcase だろうという甘い読みで引数にはしていない

	if cps.Db, err = sqlx.Open("mysql", conf.FormatDSN()); err != nil { // データベースに接続
		panic(fmt.Errorf("failed to open database: %w", err))
	}

	if _, err = cps.Db.Exec(`CREATE TABLE IF NOT EXISTS config (json JSON);`); err != nil {
		panic(fmt.Errorf("make sure your container is running!: %w", err))
		// 本来は failed to create table と返すところだけど、これに関しては原因が明らかなのでエラーメッセージを工夫
	}

	var count int // すでに存在するレコードの数
	if err := cps.Db.Get(&count, `SELECT COUNT(*) FROM config`); err != nil {
		return fmt.Errorf("failed to get count of table: %w", err)
	}

	// すでにレコードが 1 つある場合には手を加えない（レコードの数が 0 個や 2 個の異常時のみ初期化）

	if count != 1 {
		if _, err := cps.Db.Exec(`TRUNCATE TABLE config`); err != nil { // テーブルを空にする
			return fmt.Errorf("failed to truncate table %w", err)
		}

		if _, err := cps.Db.Exec(`INSERT INTO config (json) VALUES ('{}')`); err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}
	}

	log.Printf("initialized database")
	return nil
}

func Save(config any) error {
	configJson, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	// json.Marshal(config) は構造体 config を JSON 形式のテキストにする

	if _, err = cps.Db.Exec(`UPDATE config SET json = ?`, string(configJson)); err != nil {
		return fmt.Errorf("failed to update the database: %w", err)
	}
	// UPDATE文は（WHERE 以下の条件にあてはまる）全てのレコードを書き換える。レコードは一つしかないので WHERE 文は不要
	// データベース bot_db > テーブル config > レコード x 1 > フィールド json

	return nil // 必ず error 型の返り値を返す必要があるので nil を返す（nil は error 型か…？）
}

func Load(address any) error {
	var record struct { // データベースに保存されているレコードを受け取るための型
		Json string `json:"json"`
	}

	if err := cps.Db.Get(&record, "SELECT * FROM config"); err != nil {
		return fmt.Errorf("failed to get data from database: %w", err)
	}
	// record にデータベースのレコードの値を写し取って、

	if err := json.Unmarshal([]byte(record.Json), address); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	// JSON をデコードした address に参照を返す

	return nil
}
