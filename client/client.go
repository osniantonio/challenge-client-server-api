package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const URLCotacao = "http://localhost:8080/cotacao"

type ExchangeRate struct {
	Code   string `json:"code"`
	Codein string `json:"codein"`
	Name   string `json:"name"`
	Bid    string `json:"bid"`
}

func main() {
	ctx := context.Background()

	// Utilizando o package "context", o client.go terá um timeout máximo de 300ms para receber o resultado do server.go.
	ctx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URLCotacao, nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	select {
	case <-ctx.Done():
		panic("context timeout exceeded while fetching exchange rate")
	default:
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("Failed to read response body:", err)
			return
		}

		var cotacao ExchangeRate
		err = json.Unmarshal(body, &cotacao)
		if err != nil {
			fmt.Println("Failed to decode response:", err)
			return
		}

		fmt.Printf("Dólar:%v\n", cotacao)

		// O client.go terá que salvar a cotação atual em um arquivo "cotacao.txt" no formato: Dólar: {valor}
		// os.O_CREATE: Se o arquivo não existir, ele será criado. Se o arquivo já existir, a função retornará um erro.
		// os.O_WRONLY: O arquivo será aberto apenas para escrita. Se o arquivo já existir, seu conteúdo será sobrescrito.
		// os.O_TRUNC: Se o arquivo já existir, seu conteúdo será truncado (apagado)
		f, err := os.OpenFile("cotacao.txt", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		defer f.Close()

		_, err = f.WriteString(fmt.Sprintf("Dólar:%v", cotacao))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}
