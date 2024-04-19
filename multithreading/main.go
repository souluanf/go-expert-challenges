package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type ApiCepResponse struct {
	Code       string `json:"code"`
	State      string `json:"state"`
	City       string `json:"city"`
	District   string `json:"district"`
	Address    string `json:"address"`
	Status     int    `json:"status"`
	Ok         bool   `json:"ok"`
	StatusText string `json:"statusText"`
}

type ViaCepResponse struct {
	Cep         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	Uf          string `json:"uf"`
	Ibge        string `json:"ibge"`
	Gia         string `json:"gia"`
	Ddd         string `json:"ddd"`
	Siafi       string `json:"siafi"`
}

func fetchDataFromAPI(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			return "", err
		}
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Error closing response body:", err)
		}
	}(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func main() {
	var cep string
	fmt.Print("Enter a CEP: ")
	_, err := fmt.Scanf("%s", &cep)
	if err != nil {
		log.Fatalf("Error reading input: %v", err)
	}
	cep = strings.Replace(cep, "-", "", -1)
	if len(cep) != 8 {
		log.Fatalf("Invalid CEP: %s", cep)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	apiCepURL := fmt.Sprintf("https://cdn.apicep.com/file/apicep/%s.json", cep[:5]+"-"+cep[5:])
	viaCepURL := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)

	c1 := make(chan string)
	c2 := make(chan string)
	var once sync.Once

	go func() {
		data, err := fetchDataFromAPI(ctx, apiCepURL)
		if err != nil {
			c1 <- ""
			return
		}
		c1 <- data
		once.Do(cancel)
	}()

	go func() {
		data, err := fetchDataFromAPI(ctx, viaCepURL)
		if err != nil {
			c2 <- ""
			return
		}
		c2 <- data
		once.Do(cancel)
	}()

	var apiCepData ApiCepResponse
	var viaCepData ViaCepResponse

	select {
	case apiCepDataRaw := <-c1:
		err := json.Unmarshal([]byte(apiCepDataRaw), &apiCepData)
		if err != nil {
			return
		}
		fmt.Printf("Received data from CDN apicep: %+v\n", apiCepData)
	case viaCepDataRaw := <-c2:
		err := json.Unmarshal([]byte(viaCepDataRaw), &viaCepData)
		if err != nil {
			return
		}
		fmt.Printf("Received data from ViaCEP: %+v\n", viaCepData)
	case <-ctx.Done():
		fmt.Println("Timeout: Both API requests took too long.")
	}
}
