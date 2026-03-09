package python

import (
	"archive/zip"
	"bytes"
)

func PackageHandler(handlerCode string) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	fileWriter, err := zipWriter.Create("lambda_function.py")
	if err != nil {
		return nil, err
	}

	if _, err := fileWriter.Write([]byte(handlerCode)); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
