package io

import (
	"bufio"
	"io"
	"os"
)

func Save(path string, data []byte) (int, error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0766)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	l, err := file.Write(data)
	if err != nil {
		return 0, err
	}
	return l, nil
}

func Read(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	bytes := make([]byte, 0)
	r := bufio.NewReader(f)

	b := make([]byte, 512)

	for {
		n, err := r.Read(b)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err

		}
		t := b[:n]
		bytes = append(bytes, t...)
	}
	return bytes, nil
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func IsDir(path string) (bool, error) {
	s, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return s.IsDir(), nil
}
