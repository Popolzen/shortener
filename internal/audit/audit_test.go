package audit

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvent(t *testing.T) {
	before := time.Now().Unix()
	event := NewEvent(ActionShorten, "user-123", "https://example.com")
	after := time.Now().Unix()

	assert.Equal(t, ActionShorten, event.Action)
	assert.Equal(t, "user-123", event.UserID)
	assert.Equal(t, "https://example.com", event.URL)
	assert.GreaterOrEqual(t, event.Timestamp, before)
	assert.LessOrEqual(t, event.Timestamp, after)
}

func TestNewEvent_Follow(t *testing.T) {
	event := NewEvent(ActionFollow, "", "https://google.com")

	assert.Equal(t, ActionFollow, event.Action)
	assert.Empty(t, event.UserID)
}

func TestPublisher_Publish(t *testing.T) {
	pub := NewPublisher()
	mock := &mockObserver{}
	pub.Subscribe(mock)

	event := NewEvent(ActionShorten, "user-1", "https://test.com")
	pub.Publish(event)

	time.Sleep(50 * time.Millisecond) // ждём горутину

	mock.mu.Lock()
	defer mock.mu.Unlock()
	require.Len(t, mock.events, 1)
	assert.Equal(t, event.URL, mock.events[0].URL)
}

func TestPublisher_PublishMultipleObservers(t *testing.T) {
	pub := NewPublisher()
	mock1 := &mockObserver{}
	mock2 := &mockObserver{}
	pub.Subscribe(mock1)
	pub.Subscribe(mock2)

	event := NewEvent(ActionFollow, "user-2", "https://multi.com")
	pub.Publish(event)

	time.Sleep(50 * time.Millisecond)

	mock1.mu.Lock()
	assert.Len(t, mock1.events, 1)
	mock1.mu.Unlock()

	mock2.mu.Lock()
	assert.Len(t, mock2.events, 1)
	mock2.mu.Unlock()
}

func TestPublisher_Close(t *testing.T) {
	pub := NewPublisher()
	mock := &mockObserver{}
	pub.Subscribe(mock)

	err := pub.Close()

	assert.NoError(t, err)
	assert.True(t, mock.closed)
}

// Mock observer для тестов
type mockObserver struct {
	mu     sync.Mutex
	events []Event
	closed bool
}

func (m *mockObserver) Notify(event Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
}

func (m *mockObserver) Close() error {
	m.closed = true
	return nil
}

// === FileObserver tests ===

func TestFileObserver_Notify(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_test_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	obs, err := NewFileObserver(tmpFile.Name())
	require.NoError(t, err)
	defer obs.Close()

	event := NewEvent(ActionShorten, "user-123", "https://example.com")
	obs.Notify(event)

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	var parsed Event
	err = json.Unmarshal(content[:len(content)-1], &parsed) // убираем \n
	require.NoError(t, err)

	assert.Equal(t, event.URL, parsed.URL)
	assert.Equal(t, event.UserID, parsed.UserID)
	assert.Equal(t, event.Action, parsed.Action)
}

func TestFileObserver_MultipleWrites(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "audit_multi_*.log")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	obs, err := NewFileObserver(tmpFile.Name())
	require.NoError(t, err)

	obs.Notify(NewEvent(ActionShorten, "user-1", "https://one.com"))
	obs.Notify(NewEvent(ActionFollow, "user-2", "https://two.com"))
	obs.Close()

	content, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)

	lines := string(content)
	assert.Contains(t, lines, "https://one.com")
	assert.Contains(t, lines, "https://two.com")
	assert.Contains(t, lines, "shorten")
	assert.Contains(t, lines, "follow")
}

func TestFileObserver_InvalidPath(t *testing.T) {
	_, err := NewFileObserver("/nonexistent/path/audit.log")
	assert.Error(t, err)
}

// === HTTPObserver tests ===

func TestHTTPObserver_Notify(t *testing.T) {
	var received Event
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	obs := NewHTTPObserver(server.URL)
	event := NewEvent(ActionShorten, "user-http", "https://http-test.com")
	obs.Notify(event)

	assert.Equal(t, "application/json", receivedContentType)
	assert.Equal(t, event.URL, received.URL)
	assert.Equal(t, event.UserID, received.UserID)
}

func TestHTTPObserver_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	obs := NewHTTPObserver(server.URL)
	// Не должно паниковать
	obs.Notify(NewEvent(ActionFollow, "user", "https://test.com"))
}

func TestHTTPObserver_ConnectionError(t *testing.T) {
	obs := NewHTTPObserver("http://localhost:99999") // несуществующий порт
	// Не должно паниковать
	obs.Notify(NewEvent(ActionFollow, "user", "https://test.com"))
}

func TestHTTPObserver_Close(t *testing.T) {
	obs := NewHTTPObserver("http://example.com")
	err := obs.Close()
	assert.NoError(t, err)
}

// === Event JSON serialization ===

func TestEvent_JSONFormat(t *testing.T) {
	event := Event{
		Timestamp: 1234567890,
		Action:    ActionShorten,
		UserID:    "user-json",
		URL:       "https://json.com",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	expected := `{"ts":1234567890,"action":"shorten","user_id":"user-json","url":"https://json.com"}`
	assert.JSONEq(t, expected, string(data))
}

func TestEvent_JSONOmitEmptyUserID(t *testing.T) {
	event := Event{
		Timestamp: 1234567890,
		Action:    ActionFollow,
		UserID:    "",
		URL:       "https://noid.com",
	}

	data, err := json.Marshal(event)
	require.NoError(t, err)

	assert.NotContains(t, string(data), "user_id")
}
