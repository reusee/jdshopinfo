package main

import "github.com/jmoiron/sqlx"
import _ "github.com/lib/pq"

var db *sqlx.DB

func init() {
	var err error
	db, err = sqlx.Connect("postgres", "user=reus dbname=jdshopinfo sslmode=disable")
	ce(err, "connect to db")
	initSchema()
}

func initSchema() {
	db.MustExec(`CREATE TABLE IF NOT EXISTS shops (
		shop_id SERIAL PRIMARY KEY,
		name TEXT NOT NULL
	)`)
	db.MustExec(`CREATE INDEX IF NOT EXISTS shop_name ON shops (name)`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS items (
		sku BIGINT PRIMARY KEY,
		shop_id INTEGER,
		category TEXT,
		added_date TEXT
	)`)
	db.MustExec(`CREATE INDEX IF NOT EXISTS category 
		ON items (category)`)

	db.MustExec(`CREATE TABLE IF NOT EXISTS infos (
		sku BIGINT NOT NULL,
		date TEXT NOT NULL,
		good_rate SMALLINT,
		price DECIMAL(10, 2) NOT NULL,
		comments INTEGER,
		title TEXT,
		image_url TEXT
	)`)
	db.MustExec(`CREATE UNIQUE INDEX IF NOT EXISTS sku_date_info
		ON infos (sku, date)`)
	db.MustExec(`CREATE INDEX IF NOT EXISTS date
		ON infos (date)`)
}
