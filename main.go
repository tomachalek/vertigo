// Copyright 2017 Tomas Machalek <tomas.machalek@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
)

// -------------------------------------------------------

type Config struct {
	VerticalFilePath string `json:"verticalFilePath"`
	DatabasePath     string `json:"databasePath"`
	AtomStructure    string `json:"atomStructure"`
}

// -------------------------------------------------------

type MetadataCollector struct {
	conn *sql.DB // TODO
}

func (collector *MetadataCollector) process(elm Element) {
	fmt.Println("PROCESSING ELM: ", elm)
	collector.writeElement(elm)
}

func NewMetadataCollector(outputDbPath string) *MetadataCollector {
	col := MetadataCollector{}
	var err error
	col.conn, err = sql.Open("sqlite3", outputDbPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to open database: %s\n", err))
	}
	return &col
}

func (m *MetadataCollector) writeElement(elm Element) {
	rows, err := m.conn.Query("SELECT * FROM items")
	if err != nil {
		panic(fmt.Sprintf("Failed to fetch data: %s", err))
	}
	for rows.Next() {
		var id, value string
		if err := rows.Scan(&id, &value); err != nil {
			log.Fatal(err)
			continue
		}
		fmt.Println("DB ROWS: ", value)
	}
}

// -------------------------------------------------------

func loadConfig(path string) Config {
	rawData, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var conf Config
	err = json.Unmarshal(rawData, &conf)
	if err != nil {
		panic(err)
	}
	fmt.Println("CONF: ", conf)
	return conf
}

// -------------------------------------------------------

func main() {
	flag.Parse()
	if len(flag.Args()) != 1 {
		panic("Invalid arguments...")
	}
	conf := loadConfig(flag.Arg(0))
	col := NewMetadataCollector(conf.DatabasePath)
	parser := NewParser(col)
	parser.Parse(conf.VerticalFilePath)
}
