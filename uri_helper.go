package toolbox

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
)

//FileSchema file://
var FileSchema = "file://"

//ExtractMimeType extracts mime type by extension
func ExtractMimeType(file string) string {
	extension := path.Ext(file)
	if len(extension) > 1 {
		extension = extension[1:]
	}

	if mimeType, ok := FileExtensionMimeType[extension]; ok {
		return mimeType
	}
	return "text/plain"
}

//OpenReaderFromURL opens a reader from URL
func OpenReaderFromURL(rawURL string) (io.ReadCloser, string, error) {
	var url, err = url.Parse(rawURL)
	if err != nil {
		return nil, "", err
	}
	switch url.Scheme {
	case "http", "https":
		response, err := http.Get(rawURL)
		if err != nil {
			return nil, "", err
		}
		mimeType := response.Header.Get("Content-Type")
		return response.Body, mimeType, nil
	case "file":
		file, err := os.Open(url.Path)
		if err != nil {
			return nil, "", fmt.Errorf("Failed to open file: %v due to %v", rawURL, err.Error())
		}
		return file, ExtractMimeType(url.Path), nil
	}
	return nil, "", fmt.Errorf("Unsupprted url.Scheme: %v on %v", url.Scheme, rawURL)
}

//FileFromURL returns file path from passed in URL.
func FileFromURL(fileURL string) (string, error) {
	var url, err = url.Parse(fileURL)
	if err != nil {
		return "", err
	}
	switch url.Scheme {
	case "file":
		return url.Path, nil
	}
	return "", fmt.Errorf("Unsupprted url.Scheme: %v on %v", url.Scheme, fileURL)
}

//OpenURL opens passed in url as file, or error. Only file:// scheme is supported
func OpenURL(fileURL string, flag int, permissions os.FileMode) (*os.File, error) {
	filePath, err := FileFromURL(fileURL)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filePath, flag, permissions)
	if err != nil {
		return nil, fmt.Errorf("Failed to create file: %v due to %v", fileURL, err.Error())
	}
	return file, nil
}
