package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/net"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Config represents the configuration structure
type Config struct {
	ResetDay      int    `json:"reset_day"`       // 每月几号清零
	DataFile      string `json:"data_file"`       // 保存流量信息的文件路径
	LastResetDate string `json:"last_reset_date"` // 最后一次清零的日期
	Port          int    `json:"port"`            // Web 服务器监听端口
	IfName        string `json:"ifName"`
}

// TrafficData represents the traffic data structure
type TrafficData struct {
	TotalBytesSent uint64 `json:"total_bytes_sent"`
	TotalBytesRecv uint64 `json:"total_bytes_recv"`
}

var recTotal uint64 = 0
var senTotal uint64 = 0

// TrafficRecords represents the traffic records map with boot times as keys
type TrafficRecords map[string]TrafficData

// Load or create configuration from the config file
func loadOrCreateConfig(configFile string) (Config, error) {
	var config Config
	_, err := os.Stat(configFile)
	if os.IsNotExist(err) {
		// If the file doesn't exist, create it with default values
		config = Config{
			ResetDay:      1,                   // 默认每月1号清零
			DataFile:      "traffic_data.json", // 默认数据文件名
			LastResetDate: "",                  // 最后一次清零日期初始为空
			Port:          28080,               // 默认 Web 服务器监听端口
			IfName:        "eth0",              // 统计的网卡名称
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

		decoder := json.NewDecoder(transform.NewReader(file, unicode.UTF8.NewDecoder()))
		err = decoder.Decode(&config)
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
		return fmt.Errorf("error creating config file %s: %v", configFile, err)
	}
	defer file.Close()

	writer := transform.NewWriter(file, unicode.UTF8.NewEncoder())
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ") // Optional: Format JSON with indentation

	if err := encoder.Encode(config); err != nil {
		return fmt.Errorf("error encoding JSON config: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing writer: %v", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %v", err)
	}

	return nil
}

// Load or create the saved traffic data from the data file
func loadOrCreateTrafficData(dataFile string) (TrafficRecords, error) {
	var records TrafficRecords
	_, err := os.Stat(dataFile)
	if os.IsNotExist(err) {
		// If the file doesn't exist, return default data
		records = TrafficRecords{}
		err = saveTrafficData(dataFile, records)
		if err != nil {
			return records, err
		}
		log.Println("Created new traffic data file.")
	} else {
		// If the file exists, load the traffic data
		file, err := os.Open(dataFile)
		if err != nil {
			return records, err
		}
		defer file.Close()

		decoder := json.NewDecoder(transform.NewReader(file, unicode.UTF8.NewDecoder()))
		err = decoder.Decode(&records)
		if err != nil {
			return records, err
		}
	}
	return records, nil
}

// Save the current traffic data to the data file
func saveTrafficData(dataFile string, records TrafficRecords) error {
	file, err := os.Create(dataFile)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", dataFile, err)
	}
	defer file.Close()

	writer := transform.NewWriter(file, unicode.UTF8.NewEncoder())
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ") // Optional: Format JSON with indentation

	if err := encoder.Encode(records); err != nil {
		return fmt.Errorf("error encoding JSON data: %v", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("error closing writer: %v", err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("error syncing file: %v", err)
	}

	return nil
}

// Get current traffic for all interfaces
func getCurrentTraffic(ifname *string) (uint64, uint64, error) {
	interfaces, err := net.IOCounters(true)
	if err != nil {
		return 0, 0, err
	}

	var totalSent, totalRecv uint64
	for _, iface := range interfaces {
		if ifname != nil && *ifname != "" {
			if *ifname == iface.Name {
				totalSent += iface.BytesSent
				totalRecv += iface.BytesRecv
				// println("name:%s", iface.Name)
				// println("Up:%f", iface.BytesSent)
				// println("Down:%f", iface.BytesRecv)
			}
		} else {
			totalSent += iface.BytesSent
			totalRecv += iface.BytesRecv
			// println("name:%s", iface.Name)
			// println("Up:%f", iface.BytesSent)
			// println("Down:%f", iface.BytesRecv)
		}

	}

	senTotal = totalSent
	recTotal = totalRecv

	return totalSent, totalRecv, nil
}

// GetBootTime retrieves the system boot time as a string
func GetBootTime() (string, error) {
	var bootTimeStr string

	if isWindows() {
		// Use PowerShell command to get system boot time on Windows
		cmd := exec.Command("powershell", "-Command", "(Get-CimInstance -Class Win32_OperatingSystem).LastBootUpTime")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("error executing PowerShell command: %v", err)
		}
		bootTimeStr = strings.TrimSpace(string(output))
	} else {
		// Use uptime command to get system boot time on Linux
		cmd := exec.Command("uptime", "-s")
		output, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("error executing uptime command: %v", err)
		}
		bootTimeStr = strings.TrimSpace(string(output))
	}

	return bootTimeStr, nil
}

// isWindows checks if the operating system is Windows
func isWindows() bool {
	return runtime.GOOS == "windows"
}

// Check if today is the reset day and if it's time to reset the traffic
func checkAndResetTraffic(config *Config, records *TrafficRecords) error {
	// log.Println("reset reset reset reset reset reset???")

	now := time.Now()
	resetDate := time.Date(now.Year(), now.Month(), config.ResetDay, 0, 0, 0, 0, time.UTC)

	// Format the current date as a string in the format "YYYY-MM-DD"
	currentDateStr := now.Format("2006-01-02")

	layout := "2006-01-02"
	lastRestday, _ := time.Parse(layout, config.LastResetDate)

	// Check if today is the reset day and if the reset has not been done today
	if now.After(resetDate) && resetDate.After(lastRestday) {
		// Clear all traffic records
		for key := range *records {
			delete(*records, key)
		}

		// log.Println("reset reset reset reset reset reset!!!!")

		// Update the last reset date in the config
		config.LastResetDate = currentDateStr
		err := saveConfig("config.json", *config)
		if err != nil {
			return fmt.Errorf("error saving config: %v", err)
		}

		if recTotal > 0 || senTotal > 0 {
			// Get a copy of the TrafficData for the current boot time
			data, exists := (*records)["resetSum"]
			if !exists {
				data = TrafficData{}
			}
			data.TotalBytesSent = senTotal
			data.TotalBytesRecv = recTotal
			// Update the map with the modified data
			(*records)["resetSum"] = data
		}

		// Save the updated empty records
		err = saveTrafficData(config.DataFile, *records)
		if err != nil {
			return fmt.Errorf("error saving traffic data: %v", err)
		}

		log.Println("Traffic data has been reset.")
	}

	return nil
}

// Web server to handle traffic queries and return data in JSON format
func handleGetTotalTraffic(records *TrafficRecords, ifname *string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var totalSent, totalRecv, resetSent, resetRecv uint64
		for key, data := range *records {
			if key == "resetSum" {
				resetSent = data.TotalBytesSent
				resetRecv = data.TotalBytesRecv
			} else {
				totalSent += data.TotalBytesSent
				totalRecv += data.TotalBytesRecv
			}
		}
		totalSent -= resetSent
		totalRecv -= resetRecv

		// Get current traffic
		var sent, recv int64
		_sent, _recv, err := getCurrentTraffic(ifname)
		if err != nil {
			sent = -1
			recv = -1
			log.Printf("Error getting current traffic: %v", err)
		} else {
			sent = int64(_sent)
			recv = int64(_recv)
		}

		// Create a response map
		response := map[string]float64{
			"total_bytes_sent_mb":       float64(totalRecv) / 1024 / 1024,
			"total_bytes_received_mb":   float64(totalRecv) / 1024 / 1024,
			"total_bytes_sent":          float64(totalRecv),
			"total_bytes_received":      float64(totalRecv),
			"current_bytes_sent_mb":     float64(sent) / 1024 / 1024,
			"current_bytes_received_mb": float64(recv) / 1024 / 1024,
		}

		// Encode the response map to JSON and write it to the response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

func isSameDate(t1, t2 time.Time) bool {
	return t1.Year() == t2.Year() &&
		t1.Month() == t2.Month() &&
		t1.Day() == t2.Day()
}

func main() {
	// Load or create configuration
	config, err := loadOrCreateConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading or creating config: %v", err)
	}

	// Load or create saved traffic data
	records, err := loadOrCreateTrafficData(config.DataFile)
	if err != nil {
		log.Fatalf("Error loading or creating traffic data: %v", err)
	}

	// Start the web server
	http.HandleFunc("/total", handleGetTotalTraffic(&records, &config.IfName))
	addr := fmt.Sprintf(":%d", config.Port)
	go func() {
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	for {
		// Get current traffic
		sent, recv, err := getCurrentTraffic(&config.IfName)
		if err != nil {
			log.Printf("Error getting current traffic: %v", err)
			continue
		}

		// Get the system boot time as the key
		bootTime, err := GetBootTime()
		if err != nil {
			log.Printf("Error getting boot time: %v", err)
			continue
		}

		// Get a copy of the TrafficData for the current boot time
		data, exists := records[bootTime]
		if !exists {
			data = TrafficData{}
		}
		data.TotalBytesSent = sent
		data.TotalBytesRecv = recv
		// Update the map with the modified data
		records[bootTime] = data

		// log.Printf("\ntime:%s", bootTime)

		// Save the updated traffic data
		err = saveTrafficData(config.DataFile, records)
		if err != nil {
			log.Printf("Error saving traffic data: %v", err)
		}

		// Check and reset traffic data if necessary
		err = checkAndResetTraffic(&config, &records)
		if err != nil {
			log.Fatalf("Error checking and resetting traffic: %v", err)
		}

		// Print the traffic data
		// fmt.Printf("Total Bytes Sent: %.2f MB\nTotal Bytes Received: %.2f MB\n", float64(data.TotalBytesSent/1024/1024), float64(data.TotalBytesRecv/1024/1024))
		// fmt.Printf("Current Bytes Sent: %.2f MB\nCurrent Bytes Received: %.2f MB\n", float64(sent/1024/1024), float64(recv/1024/1024))

		// Wait for two seconds
		time.Sleep(3 * time.Second)
	}
}
