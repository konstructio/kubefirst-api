package common

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func GetIngressLinks(path string, domainName string) []string {
	links := []string{}

	regexPattern := regexp.MustCompile(fmt.Sprintf(`\b(?:[a-zA-Z0-9-]+\.)*%s\b[^ ]*`, domainName))

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error accessing path:", path, err)
			return err
		}
		if !info.IsDir() {
			// Process each file
			fmt.Println("Processing file:", path)
			matches := processFile(path, regexPattern)
			links = append(links, matches...)
		}
		return nil
	})

	return RemoveDuplicatesLinks(links)
}

// processFile reads the file at given path and return all links matching the domain
func processFile(filePath string, regexPattern *regexp.Regexp) []string {
	links := []string{}

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return links
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := regexPattern.FindAllString(line, -1)

		links = append(links, matches...)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}

	return links
}

// removeDuplicates returns a new slice with duplicates removed
func RemoveDuplicatesLinks(items []string) []string {
	seen := make(map[string]struct{}) // map to keep track of seen items
	var result []string

	for _, item := range items {
		if _, ok := seen[item]; !ok && !strings.Contains(item, ".git") {
			seen[item] = struct{}{} // Mark item as seen
			result = append(result, fmt.Sprintf("https://%s", item))
		}
	}

	return result
}
