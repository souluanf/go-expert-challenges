package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

func fetchDataFromAPI(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			panic(err)
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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	apiCepURL := fmt.Sprintf("https://cdn.apicep.com/file/apicep/%s.json", cep)
	viaCepURL := fmt.Sprintf("http://viacep.com.br/ws/%s/json/", cep)

	c1 := make(chan string)
	c2 := make(chan string)
	var once sync.Once

	go func() {
		data, err := fetchDataFromAPI(ctx, apiCepURL)
		if err != nil {
			log.Printf("Error fetching data from ApiCep: %v", err)
			c1 <- ""
			return
		}
		c1 <- data
		once.Do(cancel)
	}()

	go func() {
		data, err := fetchDataFromAPI(ctx, viaCepURL)
		if err != nil {
			log.Printf("Error fetching data from ViaCEP: %v", err)
			c2 <- ""
			return
		}
		c2 <- data
		once.Do(cancel)
	}()

	var apiCepData, viaCepData string
	select {
	case apiCepData = <-c1:
		fmt.Printf("Received data from CDN apicep: %s\n", apiCepData)
	case viaCepData = <-c2:
		fmt.Printf("Received data from ViaCEP: %s\n", viaCepData)
	case <-ctx.Done():
		fmt.Println("Timeout: Both API requests took too long.")
	}
}
