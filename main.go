package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shirou/gopsutil/net"
)

// Config represents the configuration structure
type Config struct {
	ResetDay int    `json:"reset_day"` // 每月几号清零
	DataFile string `json:"data_file"` // 保存流量信息的文件路径
}

// TrafficData represents the traffic data structure
type TrafficData struct {
	TotalBytesSent uint64    `json:"total_bytes_sent"`
	TotalBytesRecv uint64    `json:"total_bytes_recv"`
	LastResetDate  time.Time `json:"last_reset_date"`
}

// Load or create configuration from the config file
func loadOrCreateConfig(configFile string) (Config, error) {
	var config Config
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		// If the file doesn't exist, create it with default values
		config = Config{
			ResetDay: 1,                   // 默认每月1号清零
			DataFile: "traffic_data.json", // 默认数据文件名
		}
		err = saveConfig(configFile, config)
		if err != nil {
			return config, err
		}
		log.Println("Created default config file.")
	} else {
		// If the file exists, load the configuration
		file, err := os.Open(configFile)
		if err != nil {
			return config, err
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&config)
		if err != nil {
			return config, err
		}
	}
	return config, nil
}

// Save the current configuration to the config file
func saveConfig(configFile string, config Config) error {
	file, err := os.Create(configFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(config)
}

// Load or create the saved traffic data from the data file
func loadOrCreateTrafficData(dataFile string) (TrafficData, error) {
	var data TrafficData
	_, err := os.Stat(dataFile)
	if os.IsNotExist(err) {
		// If the file doesn't exist, return default data
		data = TrafficData{
			TotalBytesSent: 0,
			TotalBytesRecv: 0,
			LastResetDate:  time.Time{},
		}
		err = saveTrafficData(dataFile, data)
		if err != nil {
			return data, err
		}
		log.Println("Created new traffic data file.")
	} else {
		// If the file exists, load the traffic data
		file, err := os.Open(dataFile)
		if err != nil {
			return data, err
		}
		defer file.Close()

		err = json.NewDecoder(file).Decode(&data)
		if err != nil {
			return data, err
		}
	}
	return data, nil
}

// Save the current traffic data to the data file
func saveTrafficData(dataFile string, data TrafficData) error {
	file, err := os.Create(dataFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// Get current traffic for all interfaces
func getCurrentTraffic() (uint64, uint64, error) {
	interfaces, err := net.IOCounters(true)
	if err != nil {
		return 0, 0, err
	}

	var totalSent, totalRecv uint64
	for _, iface := range interfaces {
		totalSent += iface.BytesSent
		totalRecv += iface.BytesRecv
	}

	return totalSent, totalRecv, nil
}

// Check if today is the reset day and if it's time to reset the traffic
func checkAndResetTraffic(config Config, data *TrafficData) {
	now := time.Now()
	resetDate := time.Date(now.Year(), now.Month(), config.ResetDay, 0, 0, 0, 0, now.Location())

	if now.After(resetDate) && (data.LastResetDate.Before(resetDate) || data.LastResetDate.IsZero()) {
		// Reset traffic data
		data.TotalBytesSent = 0
		data.TotalBytesRecv = 0
		data.LastResetDate = now
		log.Println("Traffic data has been reset.")
	}
}

func main() {
	// Load or create configuration
	config, err := loadOrCreateConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading or creating config: %v", err)
	}

	// Load or create saved traffic data
	data, err := loadOrCreateTrafficData(config.DataFile)
	if err != nil {
		log.Fatalf("Error loading or creating traffic data: %v", err)
	}

	for {
		// Get current traffic
		sent, recv, err := getCurrentTraffic()
		if err != nil {
			log.Printf("Error getting current traffic: %v", err)
			continue
		}

		// Update total traffic data
		data.TotalBytesSent += sent
		data.TotalBytesRecv += recv

		// Check and reset traffic if necessary
		checkAndResetTraffic(config, &data)

		// Save the updated traffic data
		err = saveTrafficData(config.DataFile, data)
		if err != nil {
			log.Printf("Error saving traffic data: %v", err)
		}

		// Print the traffic data
		fmt.Printf("Total Bytes Sent: %.2f, Total Bytes Received: %.2f\n", float64(data.TotalBytesSent/1024/1024), float64(data.TotalBytesRecv/1024/1024))
		fmt.Printf("Current Bytes Sent: %.2f, Current Bytes Received: %.2f\n", float64(sent/1024/1024), float64(recv/1024/1024))

		// Wait for one minute
		time.Sleep(2 * time.Second)
	}
}
