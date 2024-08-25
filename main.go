package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"syscall"
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

func main() {
    // Fetch the free port number from the API
    port, err := getAPIResponse("http://localhost:8080/api/core/fetchfreeport")
	// fmt.Println("Plugin terminal port %d", port)
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

    // Get the user's home directory
    homeDir, err := os.UserHomeDir()
    if err != nil {
        fmt.Println("Error getting home directory:", err)
        return
    }

    // Create the command to run ttyd with bash
    args := []string{"-W", "-p", port, "--cwd", homeDir, "zsh"}
    cmd := exec.Command("ttyd", args...)
    // fmt.Printf("Running command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))

    // Connect the command's input, output, and error to the terminal's stdin, stdout, and stderr
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Ensure the terminal process is properly set up (e.g., for handling signals)
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setpgid: true,
    }

    // Run the command and check for any errors
    if err := cmd.Run(); err != nil {
        panic(err)
    }
}
