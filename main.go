package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

var (
	sessionName = "ttyd-session"
	ttydCmd     *exec.Cmd
	ttydMutex   sync.Mutex
)

// getAPIResponse sends an HTTP GET request to the specified URL and returns the response body as a string
func getAPIResponse(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// postAPIRequest sends an HTTP POST request to the specified URL with the given data
func postAPIRequest(urlStr string, data map[string]string) (string, error) {
	formData := url.Values{}
	for key, value := range data {
		formData.Set(key, value)
	}
	resp, err := http.PostForm(urlStr, formData)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// createOrAttachTmuxSession creates a new tmux session or attaches to an existing one
func createOrAttachTmuxSession() error {
	// Check if the session already exists
	checkCmd := exec.Command("tmux", "has-session", "-t", sessionName)
	if err := checkCmd.Run(); err != nil {
		// Session doesn't exist, create a new one
		createCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create tmux session: %v", err)
		}
	}
	return nil
}

// startTTYD starts the ttyd process if it's not already running
func startTTYD(port string) error {
	ttydMutex.Lock()
	defer ttydMutex.Unlock()

	if ttydCmd != nil && ttydCmd.ProcessState == nil {
		// ttyd is already running
		return nil
	}

	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("error getting home directory: %v", err)
	}

	// Create the command to run ttyd with tmux
	args := []string{"-W", "-p", port, "--cwd", homeDir, "tmux", "attach-session", "-t", sessionName, "set", "-t", sessionName ,":status off"}
	fmt.Println("ttyd " + strings.Join(args, " "))
	ttydCmd = exec.Command("ttyd", args...)
	// args = []string{}
	// fmt.Println("tmux " + strings.Join(args, " "))
	// ttydCmd = exec.Command("tmux", args...)



	// Connect the command's output and error to the os.Stdout and os.Stderr
	ttydCmd.Stdout = os.Stdout
	ttydCmd.Stderr = os.Stderr

	// Ensure the terminal process is properly set up (e.g., for handling signals)
	ttydCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Start the command
	if err := ttydCmd.Start(); err != nil {
		return fmt.Errorf("failed to start ttyd: %v", err)
	}

	go func() {
		if err := ttydCmd.Wait(); err != nil {
			log.Printf("ttyd process ended with error: %v", err)
		}
	}()

	return nil
}

func main() {
	// Create or attach to a tmux session
	if err := createOrAttachTmuxSession(); err != nil {
		log.Fatal(err)
	}

	// Fetch the free port number from the API
	port, err := getAPIResponse("http://localhost:8080/api/core/fetchfreeport")
	if err != nil {
		log.Fatal(err)
	}

	// Post the port number to the API
	data := map[string]string{
		"service": "terminal-app",
		"port":    port,
	}
	_, err = postAPIRequest("http://localhost:8080/api/core/portinsert", data)
	if err != nil {
		log.Fatal(err)
	}

	// Start ttyd
	if err := startTTYD(port); err != nil {
		log.Fatal(err)
	}

	// Keep the main goroutine running
	select {}
}