package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	ArtifactoryURL      string `json:"artifactoryUrl"`
	ArtifactoryUsername string `json:"artifactoryUsername"`
	ArtifactoryPassword string `json:"artifactoryPassword"`
}

type ArtifactoryClient struct {
	URL      string
	Username string
	Password string
	Client   *http.Client
}

type FileInfo struct {
	URI    string `json:"uri"`
	Folder bool   `json:"folder"`
	Size   int64  `json:"size"`
}

type StorageInfo struct {
	URI      string     `json:"uri"`
	Repo     string     `json:"repo"`
	Path     string     `json:"path"`
	Children []FileInfo `json:"children"`
}

type Repository struct {
	Key         string `json:"key"`
	Type        string `json:"type"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

type Module struct {
	Name            string
	RepoURL         string
	SubFolder       string // Optional: specific subfolder to build (e.g., "core" for job-engine)
	ArtifactoryRepo string // Repository name (e.g., "libs-release-local")
	ArtifactoryPath string // Full path for existence checks only (e.g., "libs-release-local/global/citytech/transaction-core")
}


// Change this Module as per your Project Structure
var modules = []Module{
	{Name: "transaction-core", RepoURL: "https://github.com/demo-repo/payment-engine.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/transaction-core"},
	{Name: "transaction-data", RepoURL: "https://github.com/demo-repo/payment-engine.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/transaction-data"},
	{Name: "caching", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/caching"},
	{Name: "configuration", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/configuration"},
	{Name: "db", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/db"},
	{Name: "iam", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/iam"},
	{Name: "iso-messaging", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/iso-messaging"},
	{Name: "job-engine", RepoURL: "https://github.com/demo-repo/job-engine.git", SubFolder: "core", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/job-engine/core"},
	{Name: "lookup", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/lookup"},
	{Name: "member-data-core", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/member-data-core"},
	{Name: "messaging", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/messaging"},
	{Name: "object-storage", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/object-storage"},
	{Name: "operator-data-core", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/operator-data-core"},
	{Name: "qr", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/qr"},
	{Name: "security", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/security"},
	{Name: "shared", RepoURL: "https://github.com/demo-repo/infrastructure.git", ArtifactoryRepo: "libs-release-local", ArtifactoryPath: "libs-release-local/global/shared"},
}

func runCommand(command string) error {
	cmd := exec.Command("powershell", "-Command", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runCommandInDir(workingDir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func NewArtifactoryClient() (*ArtifactoryClient, error) {
	url := os.Getenv("ArtifactoryUrl")
	username := os.Getenv("ArtifactoryUsername")
	password := os.Getenv("ArtifactoryPassword")

	if url == "" || username == "" || password == "" {
		configPath := "config.json"
		if len(os.Args) > 1 {
			configPath = os.Args[1]
		}
		config, err := loadConfig(configPath)
		if err == nil {
			if url == "" {
				url = config.ArtifactoryURL
			}
			if username == "" {
				username = config.ArtifactoryUsername
			}
			if password == "" {
				password = config.ArtifactoryPassword
			}
			fmt.Printf("Loaded configuration from: %s\n", configPath)
		}
	}

	if url == "" || username == "" || password == "" {
		return nil, fmt.Errorf("missing required credentials")
	}

	return &ArtifactoryClient{
		URL:      strings.TrimSuffix(url, "/"),
		Username: username,
		Password: password,
		Client:   &http.Client{},
	}, nil
}

func (ac *ArtifactoryClient) makeRequest(method, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, ac.URL+path, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(ac.Username, ac.Password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ac.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (ac *ArtifactoryClient) listRepositories() ([]Repository, error) {
	resp, err := ac.makeRequest("GET", "/api/repositories")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list repositories: %s - %s", resp.Status, string(body))
	}

	var repos []Repository
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}

	return repos, nil
}

func (ac *ArtifactoryClient) listPath(path string) (*StorageInfo, error) {
	apiPath := fmt.Sprintf("/api/storage/%s", path)
	resp, err := ac.makeRequest("GET", apiPath)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list path: %s - %s", resp.Status, string(body))
	}

	var storageInfo StorageInfo
	if err := json.NewDecoder(resp.Body).Decode(&storageInfo); err != nil {
		return nil, err
	}

	return &storageInfo, nil
}

func (ac *ArtifactoryClient) deleteItem(path string) error {
	resp, err := ac.makeRequest("DELETE", "/"+path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete item: %s - %s", resp.Status, string(body))
	}

	return nil
}

func (ac *ArtifactoryClient) createFolder(path string) error {
	// Ensure path ends with / to create a folder
	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	resp, err := ac.makeRequest("PUT", "/"+path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create folder: %s - %s", resp.Status, string(body))
	}

	return nil
}

// ---------------- UPLOAD FILE TO ARTIFACTORY ----------------
// Uploads a file to Artifactory at the specified path
func (ac *ArtifactoryClient) uploadFile(localFilePath, artifactoryPath string) error {
	file, err := os.Open(localFilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", ac.URL+"/"+artifactoryPath, file)
	if err != nil {
		return err
	}

	req.SetBasicAuth(ac.Username, ac.Password)

	// Get file info for content length
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	req.ContentLength = fileInfo.Size()

	resp, err := ac.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upload file: %s - %s", resp.Status, string(body))
	}

	return nil
}

// ---------------- FORMAT SIZE ----------------
// Converts bytes → KB / MB / GB
// Makes file size readable
// E.g., 2048 → 2.00 KB

func formatSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// hasExistingPublishing detects if build.gradle already has a publishing block
func hasExistingPublishing(content string) bool {
	// Remove comments to avoid false positives
	lines := strings.Split(content, "\n")
	var cleanedContent string
	for _, line := range lines {
		// Remove single-line comments
		if idx := strings.Index(line, "//"); idx != -1 {
			line = line[:idx]
		}
		cleanedContent += line + "\n"
	}

	// Check for publishing block or publication declarations
	return strings.Contains(cleanedContent, "publishing {") ||
		strings.Contains(cleanedContent, "publishing{") ||
		strings.Contains(cleanedContent, "MavenPublication")
}

func printMenu() {
	fmt.Println("\n=== Artifactory Manager ===")
	fmt.Println("1. List current location")
	fmt.Println("2. Navigate by number")
	// fmt.Println("3. Delete item(s)") // COMMENTED: Requires lead permission
	fmt.Println("3. Build & Publish")
	fmt.Println("4. Go back")
	fmt.Println("5. Exit")
	fmt.Print("Enter your choice: ")
}

func main() {
	client, err := NewArtifactoryClient()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Println("Provide credentials via environment variables or config.json file")
		os.Exit(1)
	}

	fmt.Println("Connected to Artifactory:", client.URL)

	scanner := bufio.NewScanner(os.Stdin)
	currentPath := ""
	pathHistory := []string{}
	var currentItems []string // Store current items (repos or folders/files)

	for {
		// Show current location
		if currentPath == "" {
			fmt.Printf("\n[Current Location: ROOT]\n")
		} else {
			fmt.Printf("\n[Current Location: /%s]\n", currentPath)
		}

		printMenu()
		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		switch choice {

		// ---------------- LIST ----------------
		// List repositories or folder contents
		case "1":
			if currentPath == "" {
				// List all repositories
				fmt.Println("\nFetching repositories...")
				repos, err := client.listRepositories()
				if err != nil {
					fmt.Printf("Error listing repositories: %v\n", err)
					continue
				}

				currentItems = []string{}
				fmt.Printf("\nRepositories (%d total)\n", len(repos))
				fmt.Println(strings.Repeat("-", 80))
				fmt.Printf("%-5s %-30s %-10s %s\n", "No.", "Repository Key", "Type", "Description")
				fmt.Println(strings.Repeat("-", 80))

				if len(repos) == 0 {
					fmt.Println("No repositories found")
				} else {
					for i, repo := range repos {
						currentItems = append(currentItems, repo.Key)
						desc := repo.Description
						if len(desc) > 35 {
							desc = desc[:32] + "..."
						}
						fmt.Printf("%-5d %-30s %-10s %s\n", i+1, repo.Key, repo.Type, desc)
					}
				}
			} else {
				// List folder contents
				storage, err := client.listPath(currentPath)
				if err != nil {
					fmt.Printf("Error listing path: %v\n", err)
					continue
				}

				currentItems = []string{}
				fmt.Printf("\nContents of: /%s\n", storage.Path)
				fmt.Println(strings.Repeat("-", 80))
				fmt.Printf("%-5s %-10s %-40s %-15s\n", "No.", "Type", "Name", "Size")
				fmt.Println(strings.Repeat("-", 80))

				if len(storage.Children) == 0 {
					fmt.Println("No items found")
				} else {
					for i, child := range storage.Children {
						name := strings.TrimPrefix(child.URI, "/")
						currentItems = append(currentItems, name)
						itemType := "FILE"
						if child.Folder {
							itemType = "FOLDER"
						}
						sizeStr := formatSize(child.Size)
						if child.Folder {
							sizeStr = "-"
						}
						fmt.Printf("%-5d %-10s %-40s %-15s\n", i+1, itemType, name, sizeStr)
					}
				}
			}

		// ---------------- NAVIGATE ----------------
		case "2":
			if len(currentItems) == 0 {
				fmt.Println("\nPlease list items first (option 1)")
				continue
			}

			fmt.Printf("\nEnter item number (1-%d): ", len(currentItems))
			scanner.Scan()
			numStr := strings.TrimSpace(scanner.Text())

			num, err := strconv.Atoi(numStr)
			if err != nil || num < 1 || num > len(currentItems) {
				fmt.Printf("Invalid number. Please enter a number between 1 and %d\n", len(currentItems))
				continue
			}

			selectedItem := currentItems[num-1]
			var newPath string
			if currentPath == "" {
				newPath = selectedItem
			} else {
				newPath = currentPath + "/" + selectedItem
			}

			_, err = client.listPath(newPath)
			if err != nil {
				fmt.Printf("Error navigating to %s: %v\n", selectedItem, err)
				continue
			}

			pathHistory = append(pathHistory, currentPath)
			currentPath = newPath
			currentItems = []string{}
			fmt.Printf("\n✓ Navigated to: /%s\n", currentPath)
			fmt.Println("Use option 1 to see contents")

		// ---------------- DELETE ----------------
		// COMMENTED OUT: Requires lead permission to use
		/*
			case "3":
				if currentPath == "" {
					fmt.Println("\nCannot delete repositories from root. Navigate inside a repository first.")
					continue
				}
				if len(currentItems) == 0 {
					fmt.Println("\nPlease list items first (option 1)")
					continue
				}

				// Delete menu
				fmt.Println("\nDelete options:")
				fmt.Println("  1. Delete single file/folder")
				fmt.Println("  2. Delete multiple files/folders by number range")
				fmt.Println("  3. Delete all versions of a major version (like 2.2.all)")
				fmt.Print("Enter your choice: ")
				scanner.Scan()
				delChoice := strings.TrimSpace(scanner.Text())

				switch delChoice {

				// Single deletion
				case "1":
					fmt.Printf("\nEnter item number to delete (1-%d): ", len(currentItems))
					scanner.Scan()
					numStr := strings.TrimSpace(scanner.Text())
					num, err := strconv.Atoi(numStr)
					if err != nil || num < 1 || num > len(currentItems) {
						fmt.Println("Invalid number")
						continue
					}
					selectedItem := currentItems[num-1]
					deletePath := currentPath + "/" + selectedItem
					fmt.Printf("Are you sure you want to delete '%s'? (yes/no): ", selectedItem)
					scanner.Scan()
					confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if confirm != "yes" {
						fmt.Println("Deletion cancelled")
						continue
					}
					err = client.deleteItem(deletePath)
					if err != nil {
						fmt.Printf("Error deleting: %v\n", err)
					} else {
						fmt.Println("✓ Deleted successfully")
						currentItems = []string{}
					}

				// Multiple deletion by range
				case "2":
					fmt.Printf("\nEnter range to delete (e.g., 1-5): ")
					scanner.Scan()
					rangeStr := strings.TrimSpace(scanner.Text())
					parts := strings.Split(rangeStr, "-")
					if len(parts) != 2 {
						fmt.Println("Invalid range format")
						continue
					}
					start, err1 := strconv.Atoi(parts[0])
					end, err2 := strconv.Atoi(parts[1])
					if err1 != nil || err2 != nil || start < 1 || end > len(currentItems) || start > end {
						fmt.Println("Invalid range numbers")
						continue
					}

					fmt.Printf("Files to be deleted:\n")
					for i := start - 1; i < end; i++ {
						fmt.Printf(" %d. %s\n", i+1, currentItems[i])
					}
					fmt.Print("Proceed? (yes/no): ")
					scanner.Scan()
					confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if confirm != "yes" {
						fmt.Println("Cancelled")
						continue
					}

					// Delete loop
					for i := start - 1; i < end; i++ {
						deletePath := currentPath + "/" + currentItems[i]
						err := client.deleteItem(deletePath)
						if err != nil {
							fmt.Printf("✗ Failed: %s\n", currentItems[i])
						} else {
							fmt.Printf("✓ Deleted: %s\n", currentItems[i])
						}
					}
					currentItems = []string{}

				// Delete all versions
				case "3":
					fmt.Print("\nEnter major version to delete (like 2.2): ")
					scanner.Scan()
					majorVer := strings.TrimSpace(scanner.Text())
					if majorVer == "" {
						fmt.Println("Invalid input")
						continue
					}

					// Filter items that match major version
					var toDelete []string
					for _, item := range currentItems {
						if strings.HasPrefix(item, majorVer+".") {
							toDelete = append(toDelete, item)
						}
					}

					if len(toDelete) == 0 {
						fmt.Println("No matching items found")
						continue
					}

					fmt.Printf("These items will be deleted:\n")
					for _, item := range toDelete {
						fmt.Println(" -", item)
					}
					fmt.Print("Proceed? (yes/no): ")
					scanner.Scan()
					confirm := strings.ToLower(strings.TrimSpace(scanner.Text()))
					if confirm != "yes" {
						fmt.Println("Cancelled")
						continue
					}

					// Delete loop
					for _, item := range toDelete {
						deletePath := currentPath + "/" + item
						err := client.deleteItem(deletePath)
						if err != nil {
							fmt.Printf("✗ Failed: %s\n", item)
						} else {
							fmt.Printf("✓ Deleted: %s\n", item)
						}
					}
					currentItems = []string{}

				default:
					fmt.Println("Invalid delete choice")
				}
		*/

		case "3":
			// Build & Publish
			fmt.Println("\n=== Build & Publish ===")
			for i, module := range modules {
				fmt.Printf("%d. %s\n", i+1, module.Name)
			}
			fmt.Printf("Enter your choice (1-%d): ", len(modules))
			scanner.Scan()
			projectChoice := strings.TrimSpace(scanner.Text())

			choiceNum, err := strconv.Atoi(projectChoice)
			if err != nil || choiceNum < 1 || choiceNum > len(modules) {
				fmt.Println("Invalid choice")
				continue
			}

			selectedModule := modules[choiceNum-1]
			projectName := selectedModule.Name
			repoURL := selectedModule.RepoURL

			// Determine Artifactory base path
			var artifactoryBasePath string
			if selectedModule.ArtifactoryPath != "" {
				// Use custom Artifactory path if specified
				artifactoryBasePath = selectedModule.ArtifactoryPath
			} else if projectName == "iso-messaging" {
				// Special case: iso-messaging must publish under iso/iso-messaging
				artifactoryBasePath = "libs-release-local/citytech/iso/iso-messaging"
			} else {
				// Default: use project name
				artifactoryBasePath = "libs-release-local/citytech/" + projectName
			}

			fmt.Printf("\n📦 Selected: %s\n", projectName)

			// Define temp directory based on repository
			var tempDir string
			if strings.Contains(repoURL, "payment-engine") {
				tempDir = "temp_payment_engine_build"
			} else if strings.Contains(repoURL, "job-engine") {
				tempDir = "temp_job_engine_build"
			} else {
				tempDir = "temp_infrastructure_build"
			}

			// Step 1: Prompt for branch name BEFORE cloning
			fmt.Print("\nEnter branch name (press Enter for default): ")
			scanner.Scan()
			branchName := strings.TrimSpace(scanner.Text())

			// Step 2: Delete existing temp directory if it exists
			if _, err := os.Stat(tempDir); err == nil {
				fmt.Printf("🧹 Removing existing temp directory: %s\n", tempDir)
				err = os.RemoveAll(tempDir)
				if err != nil {
					fmt.Printf("❌ Failed to remove temp directory: %v\n", err)
					continue
				}
				fmt.Println("✓ Temp directory removed")
			}

			// Step 3: Clone repository
			fmt.Printf("\n📥 Cloning repository from %s...\n", repoURL)
			var cloneCmd string
			if branchName != "" {
				fmt.Printf("📥 Using branch: %s\n", branchName)
				cloneCmd = fmt.Sprintf("git clone -b %s %s %s", branchName, repoURL, tempDir)
			} else {
				cloneCmd = fmt.Sprintf("git clone %s %s", repoURL, tempDir)
			}

			err = runCommand(cloneCmd)
			if err != nil {
				fmt.Printf("❌ Clone failed: %v\n", err)
				fmt.Println("Make sure git is installed and the repository URL is correct")
				if branchName != "" {
					fmt.Printf("Note: Branch '%s' may not exist\n", branchName)
				}
				continue
			}
			fmt.Println("✓ Repository cloned successfully")

			// Step 4: Navigate to project subfolder
			// Use SubFolder if specified (e.g., job-engine uses "core")
			var buildDir string
			var folderToCheck string
			if selectedModule.SubFolder != "" {
				buildDir = filepath.Join(tempDir, selectedModule.SubFolder)
				folderToCheck = selectedModule.SubFolder
			} else {
				buildDir = filepath.Join(tempDir, projectName)
				folderToCheck = projectName
			}
			if _, err := os.Stat(buildDir); os.IsNotExist(err) {
				fmt.Printf("❌ Error: Project folder '%s' not found in repository\n", folderToCheck)
				os.RemoveAll(tempDir)
				continue
			}
			fmt.Printf("✓ Found project folder: %s\n", folderToCheck)

			// Step 5: Check for Gradle wrapper
			// For multi-module projects, gradlew is at repo root, not subfolder
			var gradlewBat, gradlew string
			var gradlewDir string

			// First check at repo root
			gradlewBat = filepath.Join(tempDir, "gradlew.bat")
			gradlew = filepath.Join(tempDir, "gradlew")
			gradlewDir = tempDir

			// If not found at root, check in buildDir (for single-module projects)
			if _, err := os.Stat(gradlewBat); os.IsNotExist(err) {
				if _, err := os.Stat(gradlew); os.IsNotExist(err) {
					gradlewBat = filepath.Join(buildDir, "gradlew.bat")
					gradlew = filepath.Join(buildDir, "gradlew")
					gradlewDir = buildDir
				}
			}

			var buildTool string
			if _, err := os.Stat(gradlewBat); err == nil {
				buildTool = "gradlew.bat"
				fmt.Printf("✓ Found gradlew.bat at: %s\n", gradlewDir)
			} else if _, err := os.Stat(gradlew); err == nil {
				buildTool = "gradlew"
				fmt.Printf("✓ Found gradlew at: %s\n", gradlewDir)
				fmt.Println("❌ Error: No Gradle wrapper found (gradlew/gradlew.bat)")
				os.RemoveAll(tempDir)
				continue
			}

			// Step 6: Check for build.gradle
			buildGradle := filepath.Join(buildDir, "build.gradle")
			if _, err := os.Stat(buildGradle); os.IsNotExist(err) {
				fmt.Println("❌ Error: build.gradle not found")
				os.RemoveAll(tempDir)
				continue
			}
			fmt.Println("✓ Found build.gradle")

			// Step 7: Read version from build.gradle
			fmt.Println("\n📖 Reading version from build.gradle...")
			file, err := os.Open(buildGradle)
			if err != nil {
				fmt.Printf("❌ Failed to open build.gradle: %v\n", err)
				os.RemoveAll(tempDir)
				continue
			}

			var version string
			buildScanner := bufio.NewScanner(file)
			for buildScanner.Scan() {
				line := strings.TrimSpace(buildScanner.Text())
				// Match: version = "2.3.4.48" or version="2.3.4.48" or version = '2.3.4.48'
				if strings.HasPrefix(line, "version") && strings.Contains(line, "=") {
					// Extract version value: version = "2.3.4.48"
					parts := strings.SplitN(line, "=", 2)
					if len(parts) >= 2 {
						// Remove spaces, quotes, and trailing comments
						versionValue := strings.TrimSpace(parts[1])
						versionValue = strings.Trim(versionValue, "\"'")
						// Remove any trailing comments or semicolons
						if idx := strings.Index(versionValue, "//"); idx != -1 {
							versionValue = strings.TrimSpace(versionValue[:idx])
						}
						if idx := strings.Index(versionValue, ";"); idx != -1 {
							versionValue = strings.TrimSpace(versionValue[:idx])
						}
						version = versionValue
						break
					}
				}
			}
			file.Close()

			if version == "" {
				fmt.Println("❌ Error: Could not find version in build.gradle")
				os.RemoveAll(tempDir)
				continue
			}
			fmt.Printf("✓ Version found in build.gradle: %s\n", version)

			// Step 7b: Confirm version with user
			fmt.Printf("\n📋 Do you want to proceed with version '%s'?\n", version)
			fmt.Println("  1. Yes, proceed with this version")
			fmt.Println("  2. No, I want to change it")
			fmt.Print("Enter your choice (1 or 2): ")
			scanner.Scan()
			versionChoice := strings.TrimSpace(scanner.Text())

			if versionChoice == "2" {
				fmt.Print("\nEnter your custom version (e.g., 2.3.4.49): ")
				scanner.Scan()
				customVersion := strings.TrimSpace(scanner.Text())
				if customVersion != "" {
					version = customVersion
					fmt.Printf("✓ Using custom version: %s\n", version)
				} else {
					fmt.Println("❌ Invalid version. Using original version from build.gradle")
				}
			} else if versionChoice == "1" {
				fmt.Printf("✓ Proceeding with version: %s\n", version)
			} else {
				fmt.Println("Invalid choice. Proceeding with version from build.gradle")
			}

			// Step 8: Check if version folder already exists in Artifactory
			versionPath := artifactoryBasePath + "/" + version
			fmt.Printf("\n🔍 Checking if version folder already exists: %s\n", versionPath)
			resp, err := client.makeRequest("GET", "/api/storage/"+versionPath)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				fmt.Printf("❌ Error: Version folder '%s' already exists in Artifactory\n", version)
				fmt.Println("⚠️  Cannot overwrite existing artifacts. Build cancelled.")
				os.RemoveAll(tempDir)
				continue
			}
			if resp != nil {
				resp.Body.Close()
			}
			fmt.Println("✓ Version folder does not exist, proceeding with build")

			// Configure gradle.properties: preserve existing + add Artifactory credentials
			gradlePropsPath := filepath.Join(buildDir, "gradle.properties")
			var existingPropsContent string
			if data, err := os.ReadFile(gradlePropsPath); err == nil {
				existingPropsContent = string(data)
				if len(existingPropsContent) > 0 && !strings.HasSuffix(existingPropsContent, "\n") {
					existingPropsContent += "\n"
				}
			}

			// Check if micronautVersion is already set
			hasMicronautVersion := strings.Contains(existingPropsContent, "micronautVersion")

			// Build injected properties
			var injected string
			if !hasMicronautVersion {
				injected = fmt.Sprintf("\n# AUTO-INJECTED BY BUILD TOOL\nmicronautVersion=4.10.3\n\n# Artifactory Publishing\nartifactoryUrl=%s\nartifactoryUsername=%s\nartifactoryPassword=%s\n",
					client.URL, client.Username, client.Password)
			} else {
				injected = fmt.Sprintf("\n# Artifactory Publishing (Auto-injected)\nartifactoryUrl=%s\nartifactoryUsername=%s\nartifactoryPassword=%s\n",
					client.URL, client.Username, client.Password)
			}

			finalContent := existingPropsContent + injected

			err = os.WriteFile(gradlePropsPath, []byte(finalContent), 0644)
			if err != nil {
				fmt.Printf("❌ Failed to update gradle.properties: %v\n", err)
				os.RemoveAll(tempDir)
				continue
			}
			fmt.Println("✓ gradle.properties configured with Artifactory credentials")

			// Inject Maven publish configuration into build.gradle
			buildGradleContent, err := os.ReadFile(buildGradle)
			if err != nil {
				fmt.Printf("❌ Failed to read build.gradle: %v\n", err)
				os.RemoveAll(tempDir)
				continue
			}

			// Get the Artifactory repository name (e.g., "libs-release-local")
			artifactoryRepoName := selectedModule.ArtifactoryRepo
			if artifactoryRepoName == "" {
				artifactoryRepoName = "libs-release-local" // default
			}

			// Check if our configuration was already injected (prevent duplicate injection)
			contentStr := string(buildGradleContent)
			if strings.Contains(contentStr, "Publishing Override (Auto-injected)") {
				fmt.Println("✓ Publishing override already present, skipping injection")
			} else {
				// Inject safe publishing configuration using afterEvaluate
				// DO NOT override artifactId - let Gradle resolve it from project.name
				publishConfig := fmt.Sprintf(`

// ========== Publishing Override (Auto-injected) ==========
apply plugin: 'maven-publish'

afterEvaluate {
    publishing {
        publications {
            // Ensure mavenJava exists
            if (!findByName("mavenJava")) {
                mavenJava(MavenPublication) {
                    from components.java
                    groupId = 'global.citytech'
                    version = '%s'
                }
            }
        }

        // Override all existing repositories to prevent duplication
        repositories.clear()
        
        repositories {
            maven {
                name = "artifactory"
                url = uri("${artifactoryUrl}/%s")
                credentials {
                    username = "${artifactoryUsername}"
                    password = "${artifactoryPassword}"
                }
            }
        }
    }
}

// Disable PublishToMavenLocal to prevent duplicate publication
tasks.withType(PublishToMavenLocal).configureEach {
    enabled = false
}

// Only allow publishing mavenJava publication to Artifactory
tasks.withType(PublishToMavenRepository).configureEach {
    onlyIf { 
        publication.name == "mavenJava" && repository.name == "artifactory"
    }
}
// =========================================================
`, version, artifactoryRepoName)

				// Append to build.gradle
				buildGradleFile, err := os.OpenFile(buildGradle, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					fmt.Printf("❌ Failed to modify build.gradle: %v\n", err)
					os.RemoveAll(tempDir)
					continue
				}
				_, err = buildGradleFile.WriteString(publishConfig)
				buildGradleFile.Close()
				if err != nil {
					fmt.Printf("❌ Failed to inject publish config: %v\n", err)
					os.RemoveAll(tempDir)
					continue
				}
				fmt.Println("✓ Publishing override configured (single mavenJava publication)")
			}

			// Step 10: Build and publish the project
			fmt.Println("\n📦 Building and publishing project with Gradle...")
			var gradleWrapper string
			if strings.HasSuffix(buildTool, ".bat") {
				gradleWrapper = ".\\gradlew.bat"
			} else {
				gradleWrapper = "./gradlew"
			}

			// For multi-module projects, run from gradlewDir and specify the subproject
			var gradleArgs []string
			if selectedModule.SubFolder != "" && gradlewDir != buildDir {
				// Multi-module: run from root with project selector
				fmt.Printf("📁 Running from repo root, targeting subproject: %s\n", selectedModule.SubFolder)
				gradleArgs = []string{":" + selectedModule.SubFolder + ":clean", ":" + selectedModule.SubFolder + ":build", ":" + selectedModule.SubFolder + ":publish", "-x", "test", "--no-daemon"}
				err = runCommandInDir(gradlewDir, gradleWrapper, gradleArgs...)
			} else {
				// Single-module: run from buildDir
				gradleArgs = []string{"clean", "build", "publish", "-x", "test", "--no-daemon"}
				err = runCommandInDir(buildDir, gradleWrapper, gradleArgs...)
			}

			if err != nil {
				fmt.Printf("❌ Build or publish failed: %v\n", err)
				os.RemoveAll(tempDir)
				continue
			}
			fmt.Println("✓ Build and publish successful!")

			// Step 11: Cleanup
			fmt.Println("\n🧹 Cleaning up temporary files...")
			err = os.RemoveAll(tempDir)
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to remove temp directory: %v\n", err)
			} else {
				fmt.Println("✓ Cleanup complete!")
			}

			fmt.Println(strings.Repeat("=", 60))
			fmt.Printf("✅ Build & Publish complete!\n")
			fmt.Printf("   Project: %s\n", projectName)
			fmt.Printf("   Version: %s\n", version)
			fmt.Printf("   Published to: %s\n", artifactoryBasePath)
			fmt.Println("   Artifactory will generate: .pom, .module, .sha512, and maven-metadata.xml")
			fmt.Println(strings.Repeat("=", 60))

		case "4":
			if len(pathHistory) == 0 {
				fmt.Println("\nAlready at root")
				currentPath = ""
			} else {
				currentPath = pathHistory[len(pathHistory)-1]
				pathHistory = pathHistory[:len(pathHistory)-1]
				currentItems = []string{}
				if currentPath == "" {
					fmt.Println("\n✓ Back to ROOT")
				} else {
					fmt.Printf("\n✓ Back to: /%s\n", currentPath)
				}
				fmt.Println("Use option 1 to see contents")
			}

		case "5":
			fmt.Println("\nGoodbye!")
			os.Exit(0)

		default:
			fmt.Println("\nInvalid choice. Please try again.")
		}
	}
}
