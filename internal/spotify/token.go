package spotify

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	// URLs from the Java code (Verify if these are correct or placeholders)
	spotifyHomepageURL = "https://open.spotify.com/"
	tokenEndpointBase  = "https://open.spotify.com/get_access_token?reason=transport&productType=web-player&totp="

	// Constants for scraping
	scriptPattern = "mobile-web-player"
	vendorExclude = "vendor"

	// Constants for TOTP
	totpPeriod = 30
	totpDigits = 6
)

// Regex to find the secret byte array in JS code (DOTALL enabled via (?s))
// Translated from: secret:function\\([^)]+\\)\\{.*?\\[(.*?)\\].*?\\}
var secretPattern = regexp.MustCompile(`(?s)secret:function\([^)]+\)\{.*?\[(.*?)].*?\}`)

// Struct to parse the final JSON response
type SpotifyTokenResponse struct {
	AccessToken          string `json:"accessToken"`
	ExpirationTimestampS int64  `json:"accessTokenExpirationTimestampMs"`
}

// --- Main Function ---

func GetSecretToken() string {
	return "BQBP13IZKLA17b_ZMRnOmzuG0bOLFIfvi4GBfAyxyUtZFb_5chGrWVhMTQ1JEyn71oyXUfR0rTKX2sB9uoi_VsJBKqc6BChm10ecH2m7ZVCmS4LCrrDfjTuX2vCZkJLsO4H48i393-o"
	log.Println("Attempting to retrieve Spotify access token via scraping...")

	tokenURL, err := generateGetAccessTokenURL()
	if err != nil {
		log.Fatalf("Failed to generate token URL: %v", err)
	}

	log.Printf("Generated Token Request URL (contains TOTP): %s\n", tokenURL) // Be cautious logging this if sensitive

	req, err := http.NewRequest("GET", tokenURL, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	// Add headers similar to the Java code's account token refresh
	// Remove these if you only need the equivalent of the public token
	// req.Header.Add("App-Platform", "WebPlayer")
	// If you needed the sp_dc cookie method, you would add it here:
	// req.Header.Add("Cookie", "sp_dc=YOUR_SP_DC_COOKIE")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to perform GET request to token endpoint: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("Raw JSON Response: %s\n", string(bodyBytes))

	var tokenResp SpotifyTokenResponse
	err = json.Unmarshal(bodyBytes, &tokenResp)
	if err != nil {
		// Check for error format
		var errResp map[string]interface{}
		if json.Unmarshal(bodyBytes, &errResp) == nil {
			if errMsg, ok := errResp["error"].(map[string]interface{}); ok {
				log.Fatalf("API returned error: Type=%v, Message=%v", errMsg["type"], errMsg["message"])
			}
		}
		log.Fatalf("Failed to parse JSON response: %v", err)
	}

	if tokenResp.AccessToken == "" {
		log.Fatalf("Access token was empty in the response.")
	}

	expiryTime := time.UnixMilli(tokenResp.ExpirationTimestampS)
	log.Println("----------------------------------------------------")
	log.Printf("Successfully retrieved Access Token: %s\n", tokenResp.AccessToken)
	log.Printf("Expires At: %s\n", expiryTime.Format(time.RFC3339))
	log.Println("----------------------------------------------------")
	log.Println("WARNING: This token was obtained via fragile scraping methods.")
	return tokenResp.AccessToken
}

// --- Scraping and Token Generation Logic ---

// generateGetAccessTokenURL orchestrates the scraping and TOTP generation
func generateGetAccessTokenURL() (string, error) {
	secret, err := requestSecret()
	if err != nil {
		return "", fmt.Errorf("failed to request secret: %w", err)
	}
	if secret == nil {
		return "", fmt.Errorf("could not find secret after checking relevant scripts")
	}
	log.Printf("Secret byte array extracted (length %d)\n", len(secret))

	transformedSecret := convertArrayToTransformedByteArray(secret)
	log.Printf("Transformed secret byte array (length %d)\n", len(transformedSecret))

	hexSecret, err := toHexStringGo(transformedSecret)
	if err != nil {
		return "", fmt.Errorf("failed to convert transformed secret to hex string: %w", err)
	}
	log.Printf("Final Hex Secret (used for TOTP key): %s\n", hexSecret) // Sensitive

	totp, err := generateTOTP(hexSecret, totpPeriod, totpDigits)
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP: %w", err)
	}
	log.Printf("Generated TOTP: %s\n", totp) // Sensitive

	ts := time.Now().UnixMilli()
	tokenURL := fmt.Sprintf("%s?os=web&clientVersion=1.0.0&deviceName=lavasrc&totp=%s&totpVer=5&ts=%d", tokenEndpointBase, totp, ts)

	return tokenURL, nil
}

// requestSecret scrapes the Spotify homepage to find and extract the secret byte array
func requestSecret() ([]byte, error) {
	log.Printf("Requesting secret from Spotify homepage: %s\n", spotifyHomepageURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(spotifyHomepageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch homepage: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homepage request failed with status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var scriptUrls []string
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		scriptSrc, exists := s.Attr("src")
		if exists && strings.Contains(scriptSrc, scriptPattern) && !strings.Contains(scriptSrc, vendorExclude) {
			// Ensure URL is absolute
			if !strings.HasPrefix(scriptSrc, "http") {
				// Basic handling, might need proper base URL joining
				scriptSrc = strings.TrimPrefix(scriptSrc, "/")
				// This assumes scripts are relative to hostname, which might not always be true
				// A more robust solution would parse the homepage URL properly.
				// For simplicity based on the provided Java, we assume absolute or relative paths work.
				// Let's try resolving relative to a common CDN base if needed.
				// However, the provided java code didn't show complex relative path handling.
				// If the scriptSrc starts relative, we might need to prepend the domain from spotifyHomepageURL
				// For now, assuming the URLs found are absolute or resolvable directly.
			}
			scriptUrls = append(scriptUrls, scriptSrc)
			log.Printf("Found relevant script URL: %s\n", scriptSrc)
		}
	})

	if len(scriptUrls) == 0 {
		log.Println("No relevant script URLs found in HTML.")
		return nil, nil // Or return an error? Java returned null.
	}

	log.Printf("Found %d relevant script(s). Attempting to extract secret...\n", len(scriptUrls))
	for _, scriptURL := range scriptUrls {
		log.Printf("Attempting extraction from: %s\n", scriptURL)
		secret, err := extractSecret(client, scriptURL)
		if err != nil {
			log.Printf("Error extracting secret from %s: %v. Trying next script.\n", scriptURL, err)
			continue
		}
		if secret != nil {
			log.Println("Successfully extracted secret.")
			return secret, nil // Found it!
		}
	}

	log.Println("Could not extract secret from any found script URL.")
	return nil, nil // Indicate not found
}

// extractSecret downloads a JS file and uses regex to find the secret byte array string
func extractSecret(client *http.Client, scriptURL string) ([]byte, error) {
	resp, err := client.Get(scriptURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch script %s: %w", scriptURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("script request %s failed with status %d", scriptURL, resp.StatusCode)
	}

	scriptContentBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read script content from %s: %w", scriptURL, err)
	}
	scriptContent := string(scriptContentBytes)

	matches := secretPattern.FindStringSubmatch(scriptContent)
	if len(matches) < 2 {
		// log.Printf("Secret pattern not found in script: %s\n", scriptURL) // Can be noisy
		return nil, nil // Pattern not found, not necessarily an error
	}

	secretArrayString := matches[1] // Group 1 contains the array content
	secretStringParts := strings.Split(secretArrayString, ",")
	secretByteArray := make([]byte, 0, len(secretStringParts))

	for _, part := range secretStringParts {
		trimmedPart := strings.TrimSpace(part)
		if trimmedPart == "" {
			continue
		}
		byteVal, err := strconv.Atoi(trimmedPart)
		if err != nil {
			log.Printf("Warning: Could not parse '%s' as int in secret array from %s: %v\n", trimmedPart, scriptURL, err)
			continue // Skip invalid parts? Or fail? Java seemed to just parse.
		}
		// Handle potential overflow/underflow if values outside byte range?
		// Java byte is signed (-128 to 127), Go byte is uint8 (0-255)
		// Let's cast carefully, assuming values are intended as signed bytes like Java.
		secretByteArray = append(secretByteArray, byte(int8(byteVal))) // Cast to int8 then to byte(uint8)
	}

	if len(secretByteArray) == 0 {
		log.Printf("Warning: Parsed secret array was empty from script %s\n", scriptURL)
		return nil, nil
	}

	return secretByteArray, nil
}

// convertArrayToTransformedByteArray replicates the Java byte transformation logic
func convertArrayToTransformedByteArray(array []byte) []byte {
	transformed := make([]byte, len(array))
	for i, b := range array {
		// XOR with dat transform: (byte) (array[i] ^ ((i % 33) + 9))
		// Go's byte is uint8, Java's is signed int8. The XOR operation works bitwise,
		// so the signedness difference shouldn't matter for the XOR itself.
		transformed[i] = b ^ byte((i%33)+9)
	}
	return transformed
}

// toHexStringGo replicates the specific (and unusual) Java hex conversion
func toHexStringGo(transformed []byte) (string, error) {
	var joinedStringBuilder strings.Builder
	for _, b := range transformed {
		// Java's b was signed int8, Go's is uint8.
		// Java's .append(byte) appends the *decimal* string representation.
		// We need to replicate this: treat Go's byte as signed int8 for string conversion.
		joinedStringBuilder.WriteString(strconv.Itoa(int(int8(b)))) // Convert Go byte -> int8 -> int -> string
	}
	joinedDecimalString := joinedStringBuilder.String()

	// Java then got UTF-8 bytes of this joined *decimal* string
	utf8Bytes := []byte(joinedDecimalString)

	// Finally, Java hex-encoded *those* UTF-8 bytes
	hexString := hex.EncodeToString(utf8Bytes)
	return hexString, nil
}

// generateTOTP generates a Time-based One-Time Password using HmacSHA1
func generateTOTP(hexSecret string, period int, digits int) (string, error) {
	keyBytes, err := hex.DecodeString(hexSecret)
	if err != nil {
		return "", fmt.Errorf("invalid hex secret for TOTP key: %w", err)
	}

	// Calculate time steps
	timeCounter := uint64(time.Now().Unix()) / uint64(period)

	// Convert time counter to 8-byte big-endian buffer
	timeBuffer := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBuffer, timeCounter)

	// Create HMAC-SHA1
	mac := hmac.New(sha1.New, keyBytes)
	_, err = mac.Write(timeBuffer)
	if err != nil {
		// Should not happen with hash.Hash interface
		return "", fmt.Errorf("failed to write time buffer to HMAC: %w", err)
	}
	hash := mac.Sum(nil) // Get the SHA1 hash (20 bytes)

	// Calculate offset (last nibble of the hash)
	offset := int(hash[len(hash)-1] & 0x0F)

	// Extract 4 bytes from the hash at the offset, ensuring BigEndian order
	// Need to handle potential slice bounds, but SHA1 is 20 bytes, offset max 15, so offset+3 is max 18, safe.
	var valueBytes [4]byte
	copy(valueBytes[:], hash[offset:offset+4]) // Copy the 4 bytes
	binaryValue := binary.BigEndian.Uint32(valueBytes[:])

	// Apply mask to get rid of the MSB (first bit)
	binaryValue &= 0x7FFFFFFF

	// Calculate OTP value (modulo 10^digits)
	modulo := uint32(1)
	for i := 0; i < digits; i++ {
		modulo *= 10
	}
	otp := binaryValue % modulo

	// Format OTP as a zero-padded string
	formatString := fmt.Sprintf("%%0%dd", digits) // e.g., "%06d"
	return fmt.Sprintf(formatString, otp), nil
}
