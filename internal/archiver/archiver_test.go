package archiver

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGzipHandler_CompressResponse(t *testing.T) {
	expectedBody := "Hello, World!"

	// Хендлер, который возвращает простой текст
	handler := GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}))

	tests := []struct {
		name             string
		acceptEncoding   string
		expectCompressed bool
	}{
		{
			name:             "клиент поддерживает gzip",
			acceptEncoding:   "gzip",
			expectCompressed: true,
		},
		{
			name:             "клиент не поддерживает gzip",
			acceptEncoding:   "",
			expectCompressed: false,
		},
		{
			name:             "клиент поддерживает другие кодировки",
			acceptEncoding:   "deflate, br",
			expectCompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", tt.acceptEncoding)

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			result := rec.Result()
			defer result.Body.Close()

			// Проверяем заголовки
			if tt.expectCompressed {
				assert.Equal(t, "gzip", result.Header.Get("Content-Encoding"))
				assert.Equal(t, "text/plain", result.Header.Get("Content-Type"))

				// Декомпрессируем ответ
				gzipReader, err := gzip.NewReader(result.Body)
				require.NoError(t, err)
				defer gzipReader.Close()

				body, err := io.ReadAll(gzipReader)
				require.NoError(t, err)
				assert.Equal(t, expectedBody, string(body))
			} else {
				assert.Empty(t, result.Header.Get("Content-Encoding"))
				body, err := io.ReadAll(result.Body)
				require.NoError(t, err)
				assert.Equal(t, expectedBody, string(body))
			}
		})
	}
}

func TestGzipHandler_DecompressRequest(t *testing.T) {
	requestBody := "This is compressed request body"

	// Хендлер, который читает тело запроса
	handler := GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))

	tests := []struct {
		name               string
		contentEncoding    string
		compressRequest    bool
		expectedStatus     int
		expectDecompressed bool
	}{
		{
			name:               "запрос с gzip сжатием",
			contentEncoding:    "gzip",
			compressRequest:    true,
			expectedStatus:     http.StatusOK,
			expectDecompressed: true,
		},
		{
			name:               "запрос без сжатия",
			contentEncoding:    "",
			compressRequest:    false,
			expectedStatus:     http.StatusOK,
			expectDecompressed: false,
		},
		{
			name:               "запрос с другой кодировкой",
			contentEncoding:    "deflate",
			compressRequest:    false,
			expectedStatus:     http.StatusOK,
			expectDecompressed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody io.Reader

			if tt.compressRequest {
				// Сжимаем тело запроса
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				_, err := gzipWriter.Write([]byte(requestBody))
				require.NoError(t, err)
				gzipWriter.Close()
				reqBody = &buf
			} else {
				reqBody = bytes.NewReader([]byte(requestBody))
			}

			req := httptest.NewRequest(http.MethodPost, "/", reqBody)
			req.Header.Set("Content-Encoding", tt.contentEncoding)

			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			result := rec.Result()
			defer result.Body.Close()

			assert.Equal(t, tt.expectedStatus, result.StatusCode)

			responseBody, err := io.ReadAll(result.Body)
			require.NoError(t, err)

			if tt.expectDecompressed {
				// Ответ должен быть таким же, как исходное тело запроса
				assert.Equal(t, requestBody, string(responseBody))
			} else {
				// Ответ должен совпадать с отправленным телом (если не было сжатия)
				if !tt.compressRequest {
					assert.Equal(t, requestBody, string(responseBody))
				}
			}
		})
	}
}

func TestGzipHandler_InvalidGzipRequest(t *testing.T) {
	// Отправляем некорректные gzip данные
	invalidGzipData := []byte("this is not valid gzip data")

	handler := GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Этот хендлер не должен вызываться при ошибке декомпрессии
		t.Error("handler should not be called when gzip decompression fails")
	}))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(invalidGzipData))
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	result := rec.Result()
	defer result.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
}

func TestGzipHandler_Integration(t *testing.T) {
	// Полный интеграционный тест: сжатие запроса и ответа
	requestBody := "Hello from client"
	expectedResponse := "Hello from server"

	handler := GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Читаем декомпрессированный запрос
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, requestBody, string(body))

		// Отправляем ответ
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedResponse))
	}))

	// Сжимаем запрос
	var compressedReq bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedReq)
	_, err := gzipWriter.Write([]byte(requestBody))
	require.NoError(t, err)
	gzipWriter.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &compressedReq)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	result := rec.Result()
	defer result.Body.Close()

	// Проверяем, что ответ сжат
	assert.Equal(t, "gzip", result.Header.Get("Content-Encoding"))

	// Декомпрессируем ответ
	gzipReader, err := gzip.NewReader(result.Body)
	require.NoError(t, err)
	defer gzipReader.Close()

	responseBody, err := io.ReadAll(gzipReader)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, string(responseBody))
}
