package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hokaccha/go-prettyjson"
)

type Config struct {
	Host   string
	APIKey string
}

func InitConfig(apiKeyOpt string, hostOpt string) (*Config, error) {
	config := &Config{}

	if _, err := os.Stat(".config"); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if apiKeyOpt == "" || hostOpt == "" {
			return nil, err
		}
		config.APIKey = apiKeyOpt
		config.Host = hostOpt
		b, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(".config", b, 0o755); err != nil {
			return nil, err
		}
	} else {
		b, err := os.ReadFile(".config")
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, &config); err != nil {
			return nil, err
		}
		if apiKeyOpt != "" {
			config.APIKey = apiKeyOpt
		}
		if hostOpt != "" {
			config.Host = hostOpt
		}
		b, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(".config", b, 0o755); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func computeDataReader(dataOpt string) (io.Reader, error) {
	if dataOpt == "" {
		return nil, nil
	}

	if dataOpt[0] == '@' {
		filename, _ := strings.CutPrefix(dataOpt, "@")
		reader, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		return reader, nil
	}

	return strings.NewReader(dataOpt), nil
}

func main() {
	var usageOpt bool
	flag.BoolVar(&usageOpt, "h", false, "Print the usage and exit")
	var apiKeyOpt string
	flag.StringVar(&apiKeyOpt, "apikey", "", "Key of the API")
	var hostOpt string
	flag.StringVar(&hostOpt, "host", "", "Host to call")
	var methodOpt string
	flag.StringVar(&methodOpt, "X", "GET", "HTTP Method to call")
	var dataOpt string
	flag.StringVar(&dataOpt, "d", "", "Data to upload")
	flag.Parse()
	if usageOpt {
		flag.Usage()
		return
	}
	config, err := InitConfig(apiKeyOpt, hostOpt)
	if err != nil {
		flag.Usage()
		fmt.Printf("%s", err.Error())
		return
	}

	if config.APIKey == "" || config.Host == "" {
		flag.Usage()
		fmt.Printf("Config needed are empty: apikey='%s', host='%s'", config.APIKey, config.Host)
		return
	}
	if len(flag.Args()) != 1 {
		flag.Usage()
		fmt.Printf("Too many args: %s", flag.Args())
		return
	}
	reader, err := computeDataReader(dataOpt)
	if err != nil {
		flag.Usage()
		fmt.Printf("Invalid data: data='%s', %s", dataOpt, err.Error())
		return
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          100,
			MaxConnsPerHost:       100,
			MaxIdleConnsPerHost:   100,
		},
	}
	req, err := http.NewRequest(methodOpt, fmt.Sprintf("%s/%s", config.Host, flag.Args()[0]), reader)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.APIKey))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	result, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err.Error())
		return
	}
	result, err = prettyjson.Format(result)
	if err != nil {
		fmt.Printf("ERROR: '%s' Result not a JSON => %s\n", err.Error(), result)
		return
	}
	fmt.Println(string(result))
}
