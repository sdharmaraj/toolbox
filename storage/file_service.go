package storage

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/viant/toolbox"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"bytes"
)

var fileMode os.FileMode = 0644

//Service represents abstract way to accessing local or remote storage
type fileStorageService struct{}

func openFileFromUrl(URL string) (*os.File, error) {
	parsedUrl, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	if parsedUrl.Scheme != "file"  {
		return nil, fmt.Errorf("Invalid schema, expected file but had: %v", parsedUrl.Scheme)
	}
	return os.Open(parsedUrl.Path)
}

//List returns a list of object for supplied url
func (s *fileStorageService) List(URL string) ([]Object, error) {
	file, err := openFileFromUrl(URL)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err == nil {
		if ! stat.IsDir() {
			return []Object{
				newFileObject(URL, stat),
			}, nil
		}
	}

	files, err := file.Readdir(0)
	if err != nil {
		return nil, err
	}
	var result = make([]Object, 0)
	for _, fileInfo := range files {
		fileURL := URL + "/" + fileInfo.Name()
		result = append(result, newFileObject(fileURL,fileInfo))
	}
	return result, nil
}

//Exists returns true if resource exists
func (s *fileStorageService) Exists(URL string) (bool, error) {
	parsedUrl, err := url.Parse(URL)
	if err != nil {
		return false, err
	}
	if parsedUrl.Scheme != "file" {
		return false, fmt.Errorf("Invalid schema, expected file but had: %v", parsedUrl.Scheme)
	}
	return toolbox.FileExists(parsedUrl.Path), nil
}

//Object returns a Object for supplied url
func (s *fileStorageService) StorageObject(URL string) (Object, error) {
	file, err := openFileFromUrl(URL)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileInfo, err := os.Stat(file.Name())
	if err != nil {
		return nil, err
	}
	return newFileObject(URL, fileInfo), nil
}

//Download returns reader for downloaded storage object
func (s *fileStorageService) Download(object Object) (io.Reader, error) {
	reader, _, err := toolbox.OpenReaderFromURL(object.URL())
	if err != nil {
		defer reader.Close()
	}
	data, err :=  ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), err
}

//Upload uploads provided reader content for supplied url.
func (s *fileStorageService) Upload(URL string, reader io.Reader) error {
	parsedUrl, err := url.Parse(URL)
	if err != nil {
		return err
	}
	if parsedUrl.Scheme != "file" {
		return fmt.Errorf("Invalid schema, expected file but had: %v", parsedUrl.Scheme)
	}

	parentDir, _ := path.Split(parsedUrl.Path)

	err = toolbox.CreateDirIfNotExist(parentDir)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(parsedUrl.Path, data, fileMode)
}

func (s *fileStorageService) Register(schema string, service Service) error {
	return errors.New("unsupported")
}

//Delete removes passed in storage object
func (s *fileStorageService) Delete(object Object) error {
	fileName, err := toolbox.FileFromURL(object.URL())
	if err != nil {
		return err
	}
	return os.Remove(fileName)
}

type fileStorageObject struct {
	*AbstractObject
}

func (o *fileStorageObject) Unwrap(target interface{}) error {
	if fileInfo, casted := target.(*os.FileInfo); casted {
		source, ok := o.Source.(os.FileInfo)
		if !ok {
			return fmt.Errorf("Failed to cast %T into %T", o.Source, target)
		}
		*fileInfo = source
		return nil
	}

	return fmt.Errorf("unsuported target %T", target)
}

func (o *fileStorageObject) FileInfo() os.FileInfo {
	if source, ok := o.Source.(os.FileInfo); ok {
		return source
	}
	return nil
}


func newFileObject(url string, fileInfo os.FileInfo) Object {
	abstract := NewAbstractStorageObject(url, fileInfo , fileInfo)
	result := &fileStorageObject{
		AbstractObject: abstract,
	}
	result.AbstractObject.Object = result
	return result
}
