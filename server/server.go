package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	_ "gorm.io/driver/sqlite"
)

const URLUsdBrl = "https://economia.awesomeapi.com.br/json/last/USD-BRL"

type ExchangeRate struct {
	Code   string `json:"code"`
	Codein string `json:"codein"`
	Name   string `json:"name"`
	Bid    string `json:"bid"`
}

func main() {

	beforeStart()

	// servidor
	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", cotacaoHandler)
	http.ListenAndServe(":8080", mux)
}

func beforeStart() {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Criar a tabela se não existir
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS exchange_rate (id INTEGER PRIMARY KEY AUTOINCREMENT, bid TEXT);`)
	if err != nil {
		panic(err)
	}

	tx, err := db.Begin()
	if err != nil {
		panic(err)
	}
	if err := tx.Commit(); err != nil {
		panic(err)
	}
}

// o timeout máximo para chamar a API de cotação do dólar deverá ser de 200ms
func fetchExchangeRate(ctx context.Context) (*ExchangeRate, error) {
	client := http.Client{
		Timeout: 200 * time.Millisecond,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URLUsdBrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to fetch exchange rate. Status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]ExchangeRate
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	for _, exchangeRate := range data {
		return &exchangeRate, nil
	}

	return nil, fmt.Errorf("No exchange rate data found")
}

func findAll() {
	db, err := sql.Open("sqlite3", "./database.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT bid FROM exchange_rate;")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	for rows.Next() {
		var bidValue string
		err := rows.Scan(&bidValue)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Cotação:", bidValue)
	}
}

func saveToDatabase(ctx context.Context, db *sql.DB, exchangeRate *ExchangeRate) error {
	stmt, err := db.PrepareContext(ctx, "INSERT INTO exchange_rate (bid) VALUES (?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, exchangeRate.Bid)
	if err != nil {
		return err
	}

	return nil
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
	defer cancel()

	findAll()

	exchangeRate, err := fetchExchangeRate(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch exchange rate: %s", err), http.StatusInternalServerError)
		return
	}

	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to open database: %s", err), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// timeout máximo para conseguir persistir os dados no banco deverá ser de 10ms
	ctx, cancel = context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	err = saveToDatabase(ctx, db, exchangeRate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to save to database: %s", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"bid": exchangeRate.Bid}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal JSON response: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}
