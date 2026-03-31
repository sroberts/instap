package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dghubble/oauth1"
)

const BaseURL = "https://www.instapaper.com/api/1"

type Client struct {
	Config      *oauth1.Config
	Token       *oauth1.Token
	HTTPClient  *http.Client
}

func NewClient(consumerKey, consumerSecret, token, tokenSecret string) *Client {
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	userToken := oauth1.NewToken(token, tokenSecret)
	httpClient := config.Client(oauth1.NoContext, userToken)

	return &Client{
		Config:     config,
		Token:      userToken,
		HTTPClient: httpClient,
	}
}

// GetAccessToken exchanges a username and password for an OAuth 1.0 token.
func GetAccessToken(consumerKey, consumerSecret, username, password string) (string, string, error) {
	apiURL := "https://www.instapaper.com/api/1/oauth/access_token"
	
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	
	// Create request
	data := url.Values{}
	data.Set("x_auth_username", username)
	data.Set("x_auth_password", password)
	data.Set("x_auth_mode", "client_auth")

	// For x_auth, we sign with consumer key/secret and empty token.
	token := oauth1.NewToken("", "")
	httpClient := config.Client(oauth1.NoContext, token)

	resp, err := httpClient.PostForm(apiURL, data)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to get access token: %s (status %d)", string(body), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", "", err
	}

	return values.Get("oauth_token"), values.Get("oauth_token_secret"), nil
}

// Bookmark represents an Instapaper bookmark.
type Bookmark struct {
	ID          int    `json:"bookmark_id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	FolderID    int    `json:"folder_id"`
	Time        int64  `json:"time"`
	Starred     string `json:"starred"`
	Tags        string `json:"tags"`
}

// AddBookmark adds a new bookmark.
func (c *Client) AddBookmark(bookmarkURL string) (*Bookmark, error) {
	apiURL := BaseURL + "/bookmarks/add"
	
	data := url.Values{}
	data.Set("url", bookmarkURL)

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to add bookmark: %s (status %d)", string(body), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bookmarks []Bookmark
	if err := json.Unmarshal(body, &bookmarks); err != nil {
		return nil, err
	}

	if len(bookmarks) > 0 {
		return &bookmarks[0], nil
	}

	return nil, fmt.Errorf("no bookmark returned in response")
}

// ListBookmarks lists bookmarks in a specific folder.
func (c *Client) ListBookmarks(folderID string) ([]Bookmark, error) {
	apiURL := BaseURL + "/bookmarks/list"
	
	data := url.Values{}
	if folderID != "" {
		data.Set("folder_id", folderID)
	}

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list bookmarks: %s (status %d)", string(body), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Instapaper API returns a JSON array where the first element is the user's info,
	// and subsequent elements are bookmarks.
	var results []interface{}
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	var bookmarks []Bookmark
	for _, res := range results {
		m, ok := res.(map[string]interface{})
		if !ok {
			continue
		}
		if m["type"] == "bookmark" {
			// Re-marshal and unmarshal to convert to struct
			bBytes, _ := json.Marshal(m)
			var b Bookmark
			json.Unmarshal(bBytes, &b)
			bookmarks = append(bookmarks, b)
		}
	}

	return bookmarks, nil
}

// GetBookmarkText fetches the processed text of a bookmark (returns HTML).
func (c *Client) GetBookmarkText(id int) (string, error) {
	apiURL := BaseURL + "/bookmarks/get_text"
	
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", id))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get bookmark text: %s (status %d)", string(body), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// Folder represents an Instapaper folder.
type Folder struct {
	ID    int    `json:"folder_id"`
	Title string `json:"title"`
}

// ArchiveBookmark moves a bookmark to the archive.
func (c *Client) ArchiveBookmark(id int) error {
	apiURL := BaseURL + "/bookmarks/archive"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", id))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to archive bookmark: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// DeleteBookmark deletes a bookmark.
func (c *Client) DeleteBookmark(id int) error {
	apiURL := BaseURL + "/bookmarks/delete"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", id))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete bookmark: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// StarBookmark stars a bookmark.
func (c *Client) StarBookmark(id int) error {
	apiURL := BaseURL + "/bookmarks/star"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", id))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to star bookmark: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// UnstarBookmark unstars a bookmark.
func (c *Client) UnstarBookmark(id int) error {
	apiURL := BaseURL + "/bookmarks/unstar"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", id))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to unstar bookmark: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// MoveBookmark moves a bookmark to a specific folder.
func (c *Client) MoveBookmark(bookmarkID int, folderID int) error {
	apiURL := BaseURL + "/bookmarks/move"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", bookmarkID))
	data.Set("folder_id", fmt.Sprintf("%d", folderID))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to move bookmark: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// ListFolders lists all folders.
func (c *Client) ListFolders() ([]Folder, error) {
	apiURL := BaseURL + "/folders/list"
	
	resp, err := c.HTTPClient.PostForm(apiURL, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list folders: %s (status %d)", string(body), resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var folders []Folder
	if err := json.Unmarshal(body, &folders); err != nil {
		return nil, err
	}

	return folders, nil
}

// AddTags adds tags to a bookmark.
func (c *Client) AddTags(bookmarkID int, tags []string) error {
	apiURL := BaseURL + "/bookmarks/add_tags"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", bookmarkID))
	data.Set("tags", strings.Join(tags, ","))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add tags: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// RemoveTags removes tags from a bookmark.
func (c *Client) RemoveTags(bookmarkID int, tags []string) error {
	apiURL := BaseURL + "/bookmarks/remove_tags"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", bookmarkID))
	data.Set("tags", strings.Join(tags, ","))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to remove tags: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}

// SetTags sets the tags for a bookmark (overwrites existing tags).
func (c *Client) SetTags(bookmarkID int, tags []string) error {
	apiURL := BaseURL + "/bookmarks/set_tags"
	data := url.Values{}
	data.Set("bookmark_id", fmt.Sprintf("%d", bookmarkID))
	data.Set("tags", strings.Join(tags, ","))

	resp, err := c.HTTPClient.PostForm(apiURL, data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set tags: %s (status %d)", string(body), resp.StatusCode)
	}
	return nil
}
