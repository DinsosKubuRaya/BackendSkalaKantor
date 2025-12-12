package config

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type CloudinaryResponse struct {
	PublicID     string `json:"public_id"`
	SecureURL    string `json:"secure_url"`
	ResourceType string `json:"resource_type"`
	Format       string `json:"format"`
}

type CloudinaryErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func UploadToCloudinary(file io.Reader, fileName, folder, resourceType string) (CloudinaryResponse, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return CloudinaryResponse{}, fmt.Errorf("cloudinary credentials tidak lengkap. Pastikan CLOUDINARY_CLOUD_NAME, API_KEY, API_SECRET dan CLOUDINARY_URL sudah terisi")
	}

	fmt.Printf("☁️ Cloudinary Config loaded: cloud=%s\n", cloudName)

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload", cloudName, resourceType)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	var signatureString string
	params := []string{}

	if folder != "" {
		params = append(params, "folder="+folder)
	}

	params = append(params, "timestamp="+timestamp)
	params = append(params, "unique_filename=false")
	params = append(params, "use_filename=true")

	sort.Strings(params)

	signatureString = strings.Join(params, "&") + apiSecret

	h := sha1.New()
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return CloudinaryResponse{}, fmt.Errorf("failed to create form file: %v", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return CloudinaryResponse{}, fmt.Errorf("failed to copy file: %v", err)
	}

	writer.WriteField("api_key", apiKey)
	writer.WriteField("timestamp", timestamp)
	writer.WriteField("signature", signature)

	writer.WriteField("use_filename", "true")
	writer.WriteField("unique_filename", "false")

	if folder != "" {
		writer.WriteField("folder", folder)
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return CloudinaryResponse{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Println("📤 Sending file to Cloudinary...")

	resp, err := client.Do(req)
	if err != nil {
		return CloudinaryResponse{}, fmt.Errorf("request to Cloudinary failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	fmt.Printf("📥 Cloudinary Status: %d\n", resp.StatusCode)
	fmt.Printf("📥 Cloudinary Body: %s\n", string(respBody))

	if resp.StatusCode != 200 {
		var errResp CloudinaryErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil {
			return CloudinaryResponse{}, fmt.Errorf("cloudinary error: %s", errResp.Error.Message)
		}
		return CloudinaryResponse{}, fmt.Errorf("upload failed: %s", string(respBody))
	}

	var result CloudinaryResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return CloudinaryResponse{}, fmt.Errorf("invalid response format: %v", err)
	}

	fmt.Printf("✅ Upload success: %s\n", result.SecureURL)
	return result, nil
}

func CloudinaryFileExists(publicID, resourceType string) bool {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/upload/%s", cloudName, resourceType, publicID)

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(apiKey, apiSecret)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

func GenerateUniqueFileName(folder, originalName, resourceType string) string {
	ext := ""
	name := originalName

	if dot := strings.LastIndex(originalName, "."); dot != -1 {
		ext = originalName[dot:]
		name = originalName[:dot]
	}

	timestamp := time.Now().UnixNano()
	randomStr := uuid.New().String()[:8]
	uniqueName := fmt.Sprintf("%s_%d_%s%s", name, timestamp, randomStr, ext)

	publicID := fmt.Sprintf("%s/%s", folder, uniqueName)

	counter := 1
	for CloudinaryFileExists(publicID, resourceType) {
		newName := fmt.Sprintf("%s_%d_%s(%d)%s", name, timestamp, randomStr, counter, ext)
		publicID = fmt.Sprintf("%s/%s", folder, newName)
		counter++
	}

	return uniqueName
}

type CloudinaryDeleteResponse struct {
	Result string `json:"result"`
}

func DeleteFromCloudinary(publicID string, resourceType string) error {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return fmt.Errorf("cloudinary credentials tidak lengkap")
	}

	url := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/%s/destroy", cloudName, resourceType)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	signatureString := fmt.Sprintf("public_id=%s&timestamp=%s%s", publicID, timestamp, apiSecret)
	h := sha1.New()
	h.Write([]byte(signatureString))
	signature := hex.EncodeToString(h.Sum(nil))

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("public_id", publicID)
	writer.WriteField("api_key", apiKey)
	writer.WriteField("timestamp", timestamp)
	writer.WriteField("signature", signature)
	writer.Close()

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		return fmt.Errorf("failed to create destroy request: %v", err)
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 30 * time.Second}

	fmt.Printf("🗑️ Deleting from Cloudinary: %s (Type: %s)\n", publicID, resourceType)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request to Cloudinary destroy failed: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("🗑️ Cloudinary Delete Status: %d\n", resp.StatusCode)
	fmt.Printf("🗑️ Cloudinary Delete Body: %s\n", string(respBody))

	if resp.StatusCode != 200 {
		var errResp CloudinaryErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil {
			return fmt.Errorf("cloudinary delete error: %s", errResp.Error.Message)
		}
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	var result CloudinaryDeleteResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("invalid delete response format: %v", err)
	}

	if result.Result == "ok" || result.Result == "not found" {
		fmt.Printf("✅ Delete success (or file not found): %s\n", publicID)
		return nil
	}

	return fmt.Errorf("delete failed with result: %s", result.Result)
}
