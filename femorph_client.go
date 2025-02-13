package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

const (
	ChunkSize   = 8192
	LocalAddr   = "localhost:8000"
	ProdAddr    = "api.femorph.com"
	BasePath    = "data/cube.cdb"
	SurfacePath = "data/sphere.ply"
)

type TaskUpdate struct {
	TaskID    string  `json:"task_id"`
	Status    string  `json:"status"`
	UpdatedAt string  `json:"updatedAt"` // parsing issues
	Result    *string `json:"result,omitempty"`
	Error     *string `json:"error,omitempty"`
}

func assertHealthy(baseURL string) {
	resp, err := http.Get(fmt.Sprintf("%s/health", baseURL))
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Fatal("Application not healthy")
	}
	log.Println("Application Healthy")
}

func waitForTaskCompletion(taskID, userID, wsURL string) error {
	log.Printf("starting...")
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s/%s", wsURL, userID), nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("Waiting for task %s...", taskID)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var data map[string]interface{}
		if err := json.Unmarshal(message, &data); err != nil {
			log.Println("Invalid message format")
			continue
		}

		if data["type"] == "task_update" {
			payload, _ := json.Marshal(data["payload"])
			var update TaskUpdate

			if err := json.Unmarshal(payload, &update); err != nil {
				log.Printf("Invalid task update format: %s, error: %v", payload, err)
				continue
			}

			if update.TaskID == taskID {
				log.Printf("Task %s status: %s", taskID, update.Status)
				if update.Status == "completed" {
					return nil
				} else if update.Status == "failed" {
					return errors.New(fmt.Sprintf("Task %s failed: %v", taskID, update.Error))
				}
			}
		}
	}
}

func clearSession(userID, accessToken, baseURL string) error {

	url := fmt.Sprintf("%s/users/%s/data", baseURL, userID)
	req, err := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Printf("Cleared user session")
	return nil
}

func uploadFile(filePath, endpoint, userID, accessToken, baseURL string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", err
	}

	writer.Close()

	url := fmt.Sprintf("%s/users/%s/%s", baseURL, userID, endpoint)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Filename", filepath.Base(filePath))
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatal("Upload failed: %s", body)
	}

	var resData map[string]interface{}
	if err := json.Unmarshal(body, &resData); err != nil {
		log.Fatal("Failed to parse upload response:", err)
	}

	fileID, ok := resData["id"].(string)
	if !ok || fileID == "" {
		log.Fatal("Upload response missing file ID:", resData)
	}

	log.Printf("Upload successful: %s", fileID)
	return fileID, nil
}

func main() {
	_ = godotenv.Load()
	username := os.Getenv("FEMORPH_USERNAME")
	password := os.Getenv("FEMORPH_PASSWORD")
	baseURL := "https://" + ProdAddr
	wsURL := "wss://" + ProdAddr + "/ws/subscribe"

	// for local testing
	// baseURL := "http://" + LocalAddr
	// wsURL := "ws://" + LocalAddr + "/ws/subscribe"

	assertHealthy(baseURL)
	userID, accessToken := authenticate(username, password, baseURL)

	clearSession(userID, accessToken, baseURL)

	femID, err := uploadFile(BasePath, "fems", userID, accessToken, baseURL)
	if err != nil {
		log.Fatal(err)
	}

	surfaceID, err := uploadFile(SurfacePath, "surfaces", userID, accessToken, baseURL)
	if err != nil {
		log.Fatal(err)
	}

	taskID := morphFem(femID, surfaceID, userID, accessToken, baseURL)
	if err := waitForTaskCompletion(taskID, userID, wsURL); err != nil {
		log.Fatal(err)
	}
}

func authenticate(username, password, baseURL string) (string, string) {
	payload := map[string]string{
		"email":    username,
		"password": password,
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(fmt.Sprintf("%s/auth", baseURL), "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var data map[string]string
	json.NewDecoder(resp.Body).Decode(&data)

	var access_token = data["access_token"]
	if access_token == "" {
		log.Fatal("No access token: %s", resp)
	}
	log.Printf("Received access token for %s", username)

	return data["user_id"], access_token
}

func morphFem(femID, surfaceID, userID, accessToken, baseURL string) string {
	url := fmt.Sprintf("%s/users/%s/fems/%s/morph", baseURL, userID, femID)
	payload := map[string]string{"target": surfaceID}
	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Morph request failed: %s", body)
	}

	var resData map[string]interface{}
	if err := json.Unmarshal(body, &resData); err != nil {
		log.Fatal("Failed to parse morph response:", err)
	}

	taskID, ok := resData["task_id"].(string)
	if !ok || taskID == "" {
		log.Fatal("Morph task ID missing in response:", resData)
	}

	log.Printf("Morph task %s started", taskID)
	return taskID
}
