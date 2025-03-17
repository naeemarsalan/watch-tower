package dbconfig

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
)

// DatabaseCredentials struct holds extracted database details
type DatabaseCredentials struct {
	Name            string
	User            string
	Password        string
	Host            string
	Port            string
	WebSocketSecret string
}

// ReadDatabaseCredentials reads and extracts database credentials from the given file path
func ReadDatabaseCredentials(filePath string) (*DatabaseCredentials, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Regular expressions to extract values
	reName := regexp.MustCompile(`'NAME':\s*"([^"]+)"`)
	reUser := regexp.MustCompile(`'USER':\s*"([^"]+)"`)
	rePassword := regexp.MustCompile(`'PASSWORD':\s*"([^"]+)"`)
	reHost := regexp.MustCompile(`'HOST':\s*'([^']+)'`)
	rePort := regexp.MustCompile(`'PORT':\s*"([^"]+)"`)
	reWebSocketSecret := regexp.MustCompile(`BROADCAST_WEBSOCKET_SECRET\s*=\s*"([^"]+)"`)

	var content string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		content += scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Extract values
	credentials := &DatabaseCredentials{
		Name:            extractValue(reName, content),
		User:            extractValue(reUser, content),
		Password:        extractValue(rePassword, content),
		Host:            extractValue(reHost, content),
		Port:            extractValue(rePort, content),
		WebSocketSecret: extractValue(reWebSocketSecret, content),
	}

	return credentials, nil
}

// extractValue extracts a value from a given regex pattern
func extractValue(re *regexp.Regexp, text string) string {
	match := re.FindStringSubmatch(text)
	if len(match) > 1 {
		return match[1]
	}
	return "Not Found"
}
