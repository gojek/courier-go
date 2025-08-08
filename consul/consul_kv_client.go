package consul

import (
	"encoding/json"
	"fmt"

	consulapi "github.com/hashicorp/consul/api"
)

func GetTestConsulValue() (map[string]interface{}, error) {
	// Create Consul client for localhost
	config := consulapi.DefaultConfig()
	config.Address = "localhost:8500"

	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// Get the value for key "testConsul"
	kv := client.KV()
	pair, _, err := kv.Get("testConsul", nil)

	if err != nil {
		return nil, fmt.Errorf("failed to get key testConsul: %w", err)
	}

	if pair == nil {
		return nil, fmt.Errorf("key testConsul not found")
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal(pair.Value, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// PrintTestConsulValue gets and prints the JSON value for key "testConsul"
func PrintTestConsulValue() error {
	value, err := GetTestConsulValue()
	if err != nil {
		return err
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}

	fmt.Printf("testConsul value:\n%s\n", string(prettyJSON))

	return nil
}

// GetServiceNameFromKV extracts the serviceName from the KV store
func GetServiceNameFromKV(kvKey string) (string, error) {
	value, err := GetTestConsulValue()
	if err != nil {
		return "", err
	}

	serviceName, ok := value["serviceName"].(string)
	if !ok {
		return "", fmt.Errorf("serviceName not found in KV data")
	}

	return serviceName, nil
}
